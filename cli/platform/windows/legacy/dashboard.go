//go:build windows

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type RequestInfo struct {
	ID         uint64
	Method     string
	Path       string
	Client     string
	StartedAt  time.Time
	FinishedAt time.Time
	DurationMs int64
	Status     int
	BytesIn    int64
	BytesOut   int64
	Error      string
}

type GPUInfo struct {
	Available   bool
	Name        string
	Utilization float64
	MemoryUsed  float64
	MemoryTotal float64
	Temperature float64
	PowerWatts  float64
	Error       string
}

type ModelInfo struct {
	Name          string
	Size          int64
	SizeVRAM      int64
	ContextLength int64
	ExpiresAt     string
}

type PowerInfo struct {
	Available        bool
	PluggedIn        bool
	Charging         bool
	BatterySaver     bool
	Percent          int
	RemainingSeconds int64
	State            string
}
type SystemInfo struct {
	Hostname       string
	OS             string
	Architecture   string
	CPUs           int
	CPUPercent     float64
	MemoryUsed     uint64
	MemoryTotal    uint64
	UptimeSeconds  uint64
	LANAddresses   []string
	GPU            GPUInfo
	Power          PowerInfo
	Models         []ModelInfo
	BackendHealthy bool
	BackendVersion string
	GatewayStarted time.Time
}

type MetricsSnapshot struct {
	UpdatedAt      time.Time
	System         SystemInfo
	Active         []RequestInfo
	Recent         []RequestInfo
	TotalRequests  uint64
	FailedRequests uint64
	CompletedBytes int64
	AverageLatency int64
	ListenAddress  string
	BackendAddress string
}

type metricStore struct {
	mu             sync.RWMutex
	system         SystemInfo
	active         map[uint64]*RequestInfo
	recent         []RequestInfo
	total          uint64
	failed         uint64
	completedBytes int64
	totalLatency   int64
	listen         string
	backend        string
}

func newMetricStore(listen, backend string) *metricStore {
	host, _ := os.Hostname()
	return &metricStore{
		system: SystemInfo{
			Hostname:       host,
			OS:             runtime.GOOS,
			Architecture:   runtime.GOARCH,
			CPUs:           runtime.NumCPU(),
			LANAddresses:   lanAddresses(),
			GatewayStarted: time.Now(),
		},
		active:  make(map[uint64]*RequestInfo),
		listen:  listen,
		backend: backend,
	}
}

func (s *metricStore) begin(r *http.Request) uint64 {
	id := atomic.AddUint64(&requestID, 1)
	client := r.RemoteAddr
	if host, _, err := net.SplitHostPort(client); err == nil {
		client = host
	}
	item := &RequestInfo{
		ID:        id,
		Method:    r.Method,
		Path:      r.URL.Path,
		Client:    client,
		StartedAt: time.Now(),
		BytesIn:   r.ContentLength,
	}
	s.mu.Lock()
	s.active[id] = item
	s.total++
	s.mu.Unlock()
	return id
}

func (s *metricStore) setStatus(id uint64, status int) {
	s.mu.Lock()
	if item := s.active[id]; item != nil {
		item.Status = status
	}
	s.mu.Unlock()
}

func (s *metricStore) setError(id uint64, err error) {
	s.mu.Lock()
	if item := s.active[id]; item != nil {
		item.Error = err.Error()
	}
	s.mu.Unlock()
}

func (s *metricStore) finish(id uint64, status int, bytesOut int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.active[id]
	if item == nil {
		return
	}
	delete(s.active, id)
	item.FinishedAt = time.Now()
	item.DurationMs = item.FinishedAt.Sub(item.StartedAt).Milliseconds()
	if status != 0 {
		item.Status = status
	}
	if item.Status == 0 {
		item.Status = 200
	}
	item.BytesOut = bytesOut
	if item.Status >= 400 || item.Error != "" {
		s.failed++
	}
	s.completedBytes += item.BytesOut
	s.totalLatency += item.DurationMs
	s.recent = append([]RequestInfo{*item}, s.recent...)
	if len(s.recent) > 12 {
		s.recent = s.recent[:12]
	}
}

func (s *metricStore) snapshot() MetricsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	active := make([]RequestInfo, 0, len(s.active))
	for _, item := range s.active {
		copyItem := *item
		copyItem.DurationMs = time.Since(copyItem.StartedAt).Milliseconds()
		active = append(active, copyItem)
	}
	sort.Slice(active, func(i, j int) bool { return active[i].StartedAt.Before(active[j].StartedAt) })
	recent := append([]RequestInfo(nil), s.recent...)
	average := int64(0)
	completed := s.total - uint64(len(s.active))
	if completed > 0 {
		average = s.totalLatency / int64(completed)
	}
	return MetricsSnapshot{
		UpdatedAt:      time.Now(),
		System:         s.system,
		Active:         active,
		Recent:         recent,
		TotalRequests:  s.total,
		FailedRequests: s.failed,
		CompletedBytes: s.completedBytes,
		AverageLatency: average,
		ListenAddress:  s.listen,
		BackendAddress: s.backend,
	}
}

func (s *metricStore) updateSystem(info SystemInfo) {
	s.mu.Lock()
	info.GatewayStarted = s.system.GatewayStarted
	s.system = info
	s.mu.Unlock()
}

var requestID uint64

type countingWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *countingWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *countingWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += int64(n)
	return n, err
}

func (w *countingWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *countingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacking not supported")
	}
	return h.Hijack()
}

func (w *countingWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := w.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func serveGateway(listen, backend string) error {
	target, err := url.Parse(backend)
	if err != nil {
		return err
	}
	store := newMetricStore(listen, backend)
	go collectSystemLoop(store, target)

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Host = target.Host
	}
	proxy.FlushInterval = -1
	proxy.ModifyResponse = func(resp *http.Response) error {
		if id, ok := resp.Request.Context().Value(requestContextKey{}).(uint64); ok {
			store.setStatus(id, resp.StatusCode)
		}
		return nil
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		if id, ok := r.Context().Value(requestContextKey{}).(uint64); ok {
			store.setError(id, proxyErr)
		}
		http.Error(w, "Ollama backend unavailable", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/_myference/metrics", func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackRequest(r) {
			http.Error(w, "loopback only", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(store.snapshot())
	})
	mux.HandleFunc("/_myference/health", func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackRequest(r) {
			http.Error(w, "loopback only", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "{\"status\":\"ok\"}")
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id := store.begin(r)
		cw := &countingWriter{ResponseWriter: w}
		ctx := context.WithValue(r.Context(), requestContextKey{}, id)
		proxy.ServeHTTP(cw, r.WithContext(ctx))
		store.finish(id, cw.status, cw.bytes)
	})

	server := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       5 * time.Minute,
	}
	fmt.Printf("Myference gateway listening on %s -> %s\n", listen, backend)
	return server.ListenAndServe()
}

type requestContextKey struct{}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
	procGetTickCount64       = kernel32.NewProc("GetTickCount64")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
)

func memoryInfo() (uint64, uint64) {
	var status memoryStatusEx
	status.Length = uint32(unsafe.Sizeof(status))
	ok, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return 0, 0
	}
	return status.TotalPhys - status.AvailPhys, status.TotalPhys
}

func queryPower() PowerInfo {
	var status systemPowerStatus
	ok, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return PowerInfo{Percent: -1, RemainingSeconds: -1, State: "unavailable"}
	}
	info := PowerInfo{
		Available:        status.BatteryFlag&128 == 0,
		PluggedIn:        status.ACLineStatus == 1,
		Charging:         status.BatteryFlag&8 != 0,
		BatterySaver:     status.SystemStatusFlag == 1,
		Percent:          int(status.BatteryLifePercent),
		RemainingSeconds: int64(status.BatteryLifeTime),
	}
	if status.BatteryLifePercent == 255 {
		info.Percent = -1
	}
	if status.BatteryLifeTime == ^uint32(0) {
		info.RemainingSeconds = -1
	}
	switch {
	case !info.Available:
		info.State = "no battery"
	case status.BatteryFlag&4 != 0:
		info.State = "critical"
	case info.Charging:
		info.State = "charging"
	case info.PluggedIn:
		info.State = "plugged in"
	default:
		info.State = "discharging"
	}
	return info
}

type cpuSample struct {
	idle   uint64
	kernel uint64
	user   uint64
	valid  bool
}

func filetimeValue(ft syscall.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

func sampleCPU(previous cpuSample) (float64, cpuSample) {
	var idleFT, kernelFT, userFT syscall.Filetime
	ok, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleFT)),
		uintptr(unsafe.Pointer(&kernelFT)),
		uintptr(unsafe.Pointer(&userFT)),
	)
	if ok == 0 {
		return 0, previous
	}
	current := cpuSample{
		idle:   filetimeValue(idleFT),
		kernel: filetimeValue(kernelFT),
		user:   filetimeValue(userFT),
		valid:  true,
	}
	if !previous.valid {
		return 0, current
	}
	idleDelta := current.idle - previous.idle
	totalDelta := (current.kernel - previous.kernel) + (current.user - previous.user)
	if totalDelta == 0 {
		return 0, current
	}
	usage := 100 * (1 - float64(idleDelta)/float64(totalDelta))
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}
	return usage, current
}

func uptimeSeconds() uint64 {
	value, _, _ := procGetTickCount64.Call()
	return uint64(value) / 1000
}

func lanAddresses() []string {
	var result []string
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		lower := strings.ToLower(iface.Name)
		if strings.Contains(lower, "vmware") || strings.Contains(lower, "virtual") ||
			strings.Contains(lower, "vethernet") || strings.Contains(lower, "wsl") ||
			strings.Contains(lower, "docker") {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() {
				result = append(result, ip4.String())
			}
		}
	}
	sort.Strings(result)
	return result
}

func queryGPU() GPUInfo {
	command := exec.Command("nvidia-smi.exe",
		"--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits")
	output, err := command.Output()
	if err != nil {
		return GPUInfo{Available: false, Error: compactError(err)}
	}
	line := strings.TrimSpace(strings.Split(string(output), "\n")[0])
	parts := strings.Split(line, ",")
	if len(parts) < 6 {
		return GPUInfo{Available: false, Error: "unexpected nvidia-smi output"}
	}
	number := func(value string) float64 {
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return parsed
	}
	return GPUInfo{
		Available:   true,
		Name:        strings.TrimSpace(parts[0]),
		Utilization: number(parts[1]),
		MemoryUsed:  number(parts[2]),
		MemoryTotal: number(parts[3]),
		Temperature: number(parts[4]),
		PowerWatts:  number(parts[5]),
	}
}

func compactError(err error) string {
	text := strings.ReplaceAll(err.Error(), "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > 80 {
		return text[:80]
	}
	return text
}

func queryBackend(target *url.URL) (bool, string, []ModelInfo) {
	client := &http.Client{Timeout: 2 * time.Second}
	version := ""
	healthy := false
	if response, err := client.Get(target.String() + "/api/version"); err == nil {
		defer response.Body.Close()
		var payload map[string]interface{}
		if json.NewDecoder(response.Body).Decode(&payload) == nil {
			version, _ = payload["version"].(string)
		}
		healthy = response.StatusCode == http.StatusOK
	}

	var models []ModelInfo
	if response, err := client.Get(target.String() + "/api/ps"); err == nil {
		defer response.Body.Close()
		var payload map[string]interface{}
		if json.NewDecoder(response.Body).Decode(&payload) == nil {
			if entries, ok := payload["models"].([]interface{}); ok {
				for _, entry := range entries {
					item, _ := entry.(map[string]interface{})
					model := ModelInfo{}
					model.Name, _ = item["name"].(string)
					model.Size = jsonNumber(item["size"])
					model.SizeVRAM = jsonNumber(item["size_vram"])
					model.ContextLength = jsonNumber(item["context_length"])
					model.ExpiresAt, _ = item["expires_at"].(string)
					models = append(models, model)
				}
			}
		}
	}
	return healthy, version, models
}

func jsonNumber(value interface{}) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		result, _ := typed.Int64()
		return result
	default:
		return 0
	}
}

func collectSystemLoop(store *metricStore, target *url.URL) {
	var previous cpuSample
	for {
		cpu, current := sampleCPU(previous)
		previous = current
		used, total := memoryInfo()
		power := queryPower()
		gpu := queryGPU()
		healthy, version, models := queryBackend(target)
		host, _ := os.Hostname()
		store.updateSystem(SystemInfo{
			Hostname:       host,
			OS:             runtime.GOOS,
			Architecture:   runtime.GOARCH,
			CPUs:           runtime.NumCPU(),
			CPUPercent:     cpu,
			MemoryUsed:     used,
			MemoryTotal:    total,
			UptimeSeconds:  uptimeSeconds(),
			LANAddresses:   lanAddresses(),
			GPU:            gpu,
			Power:          power,
			Models:         models,
			BackendHealthy: healthy,
			BackendVersion: version,
		})
		time.Sleep(2 * time.Second)
	}
}

func viewDashboard(metricsURL string) error {
	enableRawConsole()
	defer restoreConsole()
	keyChannel := make(chan byte, 1)
	go func() {
		buffer := make([]byte, 1)
		for {
			if _, err := os.Stdin.Read(buffer); err != nil {
				return
			}
			keyChannel <- buffer[0]
		}
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case key := <-keyChannel:
			if key == 'q' || key == 'Q' || key == 3 {
				fmt.Print("\x1b[0m\x1b[2J\x1b[H")
				return nil
			}
		case <-ticker.C:
			response, err := client.Get(metricsURL)
			if err != nil {
				renderUnavailable(err)
				continue
			}
			var snapshot MetricsSnapshot
			decodeErr := json.NewDecoder(response.Body).Decode(&snapshot)
			response.Body.Close()
			if decodeErr != nil {
				renderUnavailable(decodeErr)
				continue
			}
			renderDashboard(snapshot)
		}
	}
}

var (
	consoleMode     uint32
	consoleHandle   syscall.Handle
	consoleModified bool
)

func enableRawConsole() {
	procGetStdHandle := kernel32.NewProc("GetStdHandle")
	procGetConsoleMode := kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode := kernel32.NewProc("SetConsoleMode")
	stdInputHandle := int32(-10)
	handle, _, _ := procGetStdHandle.Call(uintptr(stdInputHandle))
	consoleHandle = syscall.Handle(handle)
	if ok, _, _ := procGetConsoleMode.Call(handle, uintptr(unsafe.Pointer(&consoleMode))); ok != 0 {
		newMode := consoleMode &^ uint32(0x0002) &^ uint32(0x0004)
		if ok, _, _ := procSetConsoleMode.Call(handle, uintptr(newMode)); ok != 0 {
			consoleModified = true
		}
	}
}

func restoreConsole() {
	if consoleModified {
		procSetConsoleMode := kernel32.NewProc("SetConsoleMode")
		procSetConsoleMode.Call(uintptr(consoleHandle), uintptr(consoleMode))
	}
}

func renderUnavailable(err error) {
	fmt.Print("\x1b[2J\x1b[H")
	fmt.Println("\x1b[36;1mMYFERENCE DASHBOARD\x1b[0m")
	fmt.Println()
	fmt.Println("\x1b[31mWaiting for gateway metrics:\x1b[0m", compactError(err))
	fmt.Println()
	fmt.Println("Press Q to stop.")
}

func renderDashboard(s MetricsSnapshot) {
	fmt.Print("\x1b[2J\x1b[H")
	statusColor := "\x1b[32m"
	statusText := "HEALTHY"
	if !s.System.BackendHealthy {
		statusColor = "\x1b[31m"
		statusText = "BACKEND DOWN"
	}
	fmt.Printf("\x1b[36;1mMYFERENCE INFERENCE SERVER\x1b[0m  %s%s\x1b[0m  %s\n",
		statusColor, statusText, time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println(strings.Repeat("=", 96))
	endpoints := make([]string, 0, len(s.System.LANAddresses))
	_, port, _ := net.SplitHostPort(s.ListenAddress)
	for _, address := range s.System.LANAddresses {
		endpoints = append(endpoints, "http://"+address+":"+port)
	}
	fmt.Printf("Host %-20s  OS %-12s  CPU %2d cores  Uptime %s\n",
		s.System.Hostname, s.System.OS+"/"+s.System.Architecture, s.System.CPUs, formatDuration(time.Duration(s.System.UptimeSeconds)*time.Second))
	fmt.Printf("API  %-68s  Ollama %s\n", strings.Join(endpoints, "  "), emptyDash(s.System.BackendVersion))
	ramPercent := percent(s.System.MemoryUsed, s.System.MemoryTotal)
	fmt.Printf("CPU  %s %5.1f%%    RAM %s %5.1f%%  %s / %s\n",
		bar(s.System.CPUPercent, 20), s.System.CPUPercent,
		bar(ramPercent, 20), ramPercent,
		formatBytes(int64(s.System.MemoryUsed)), formatBytes(int64(s.System.MemoryTotal)))
	if ramPercent >= 95 {
		fmt.Println("\x1b[31;1mWARNING: critical RAM pressure; paging will severely reduce inference speed.\x1b[0m")
	}

	power := s.System.Power
	powerSource := "Battery"
	if power.PluggedIn {
		powerSource = "AC power"
	}
	battery := "not detected"
	if power.Available {
		battery = power.State
		if power.Percent >= 0 {
			battery = fmt.Sprintf("%d%% %s", power.Percent, power.State)
		}
		if !power.PluggedIn && power.RemainingSeconds >= 0 {
			battery += " (" + formatDuration(time.Duration(power.RemainingSeconds)*time.Second) + " remaining)"
		}
		if power.BatterySaver {
			battery += " | saver on"
		}
	}
	fmt.Printf("Power %-12s  %s\n", powerSource, battery)
	gpu := s.System.GPU
	if gpu.Available {
		fmt.Printf("GPU  %-31s SM %s %5.1f%%  VRAM %6.0f/%-6.0f MiB  %3.0f C  %5.1f W\n",
			trim(gpu.Name, 31), bar(gpu.Utilization, 14), gpu.Utilization,
			gpu.MemoryUsed, gpu.MemoryTotal, gpu.Temperature, gpu.PowerWatts)
	} else {
		fmt.Printf("GPU  \x1b[31mUnavailable\x1b[0m  %s\n", emptyDash(gpu.Error))
	}

	fmt.Println(strings.Repeat("-", 96))
	if len(s.System.Models) == 0 {
		fmt.Println("Models: none currently loaded")
	} else {
		fmt.Print("Models:")
		for _, model := range s.System.Models {
			placement := ""
			if model.Size > 0 {
				gpuPercent := 100 * float64(model.SizeVRAM) / float64(model.Size)
				placement = fmt.Sprintf(" VRAM %.0f%%", gpuPercent)
				if gpuPercent < 95 {
					placement += fmt.Sprintf(" \x1b[31;1mCPU OFFLOAD %.0f%%\x1b[0m", 100-gpuPercent)
				}
			}
			contextText := ""
			if model.ContextLength > 0 {
				contextText = fmt.Sprintf(" ctx %d", model.ContextLength)
			}
			fmt.Printf("  \x1b[35m%s\x1b[0m (%s%s%s)", model.Name, formatBytes(model.Size), placement, contextText)
		}
		fmt.Println()
	}

	fmt.Printf("Requests  total %-6d active %-3d failed %-4d avg %-8s transferred %s\n",
		s.TotalRequests, len(s.Active), s.FailedRequests,
		formatMilliseconds(s.AverageLatency), formatBytes(s.CompletedBytes))
	fmt.Println(strings.Repeat("-", 96))
	fmt.Println("\x1b[33;1mACTIVE REQUESTS\x1b[0m")
	if len(s.Active) == 0 {
		fmt.Println("  Idle - waiting for a client request")
	} else {
		for _, request := range s.Active {
			fmt.Printf("  #%-4d %-15s %-5s %-38s %9s  in %s\n",
				request.ID, trim(request.Client, 15), request.Method, trim(request.Path, 38),
				formatMilliseconds(request.DurationMs), formatBytes(request.BytesIn))
		}
	}

	fmt.Println(strings.Repeat("-", 96))
	fmt.Println("\x1b[34;1mRECENT REQUESTS\x1b[0m")
	if len(s.Recent) == 0 {
		fmt.Println("  No completed requests yet")
	} else {
		for _, request := range s.Recent {
			status := fmt.Sprintf("%d", request.Status)
			if request.Status >= 400 {
				status = "\x1b[31m" + status + "\x1b[0m"
			} else {
				status = "\x1b[32m" + status + "\x1b[0m"
			}
			fmt.Printf("  %-8s %-15s %-5s %-34s %s %9s %9s\n",
				request.FinishedAt.Format("15:04:05"), trim(request.Client, 15), request.Method,
				trim(request.Path, 34), status, formatMilliseconds(request.DurationMs), formatBytes(request.BytesOut))
		}
	}
	fmt.Println(strings.Repeat("=", 96))
	fmt.Println("Press \x1b[1mQ\x1b[0m to close this dashboard (start/headless will also stop Myference).")
}

func trim(value string, width int) string {
	if len(value) <= width {
		return value
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "~"
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(used) / float64(total)
}

func bar(value float64, width int) string {
	filled := int(value / 100 * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func formatBytes(value int64) string {
	if value < 0 {
		return "-"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	number := float64(value)
	unit := 0
	for number >= 1024 && unit < len(units)-1 {
		number /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", value, units[unit])
	}
	return fmt.Sprintf("%.1f %s", number, units[unit])
}

func formatMilliseconds(value int64) string {
	if value < 1000 {
		return fmt.Sprintf("%dms", value)
	}
	return (time.Duration(value) * time.Millisecond).Round(time.Millisecond).String()
}

func formatDuration(value time.Duration) string {
	value = value.Round(time.Second)
	days := value / (24 * time.Hour)
	value %= 24 * time.Hour
	if days > 0 {
		return fmt.Sprintf("%dd %s", days, value)
	}
	return value.String()
}

func main() {
	serve := flag.Bool("serve", false, "run the inference gateway")
	view := flag.Bool("view", false, "show the live dashboard")
	listen := flag.String("listen", "0.0.0.0:11434", "gateway listen address")
	backend := flag.String("backend", "http://127.0.0.1:11435", "Ollama backend URL")
	metrics := flag.String("metrics", "http://127.0.0.1:11434/_myference/metrics", "dashboard metrics URL")
	flag.Parse()

	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	_ = signalContext

	switch {
	case *serve:
		if err := serveGateway(*listen, *backend); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *view:
		if err := viewDashboard(*metrics); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Println("Usage:")
		fmt.Println("  myference-dashboard.exe -serve [-listen 0.0.0.0:11434] [-backend http://127.0.0.1:11435]")
		fmt.Println("  myference-dashboard.exe -view [-metrics http://127.0.0.1:11434/_myference/metrics]")
	}
}
