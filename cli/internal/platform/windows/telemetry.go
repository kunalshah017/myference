package windows

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/kunalshah017/myference/cli/internal/provider"
)

type NVIDIAGPU struct {
	Name               string `json:"name"`
	UtilizationPercent int    `json:"utilizationPercent"`
	MemoryUsedMiB      int    `json:"memoryUsedMiB"`
	MemoryTotalMiB     int    `json:"memoryTotalMiB"`
	TemperatureC       int    `json:"temperatureC"`
}

type HostTelemetry struct {
	CPUPercent       float64       `json:"cpuPercent"`
	MemoryUsedBytes  uint64        `json:"memoryUsedBytes"`
	MemoryTotalBytes uint64        `json:"memoryTotalBytes"`
	OnACPower        bool          `json:"onACPower"`
	BatteryPercent   int           `json:"batteryPercent"`
	GPUs             []NVIDIAGPU   `json:"gpus,omitempty"`
	LoadedModels     []LoadedModel `json:"loadedModels,omitempty"`
}

func ParseHostTelemetry(input io.Reader) (HostTelemetry, error) {
	var telemetry HostTelemetry
	decoder := json.NewDecoder(io.LimitReader(input, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&telemetry); err != nil {
		return HostTelemetry{}, err
	}
	if telemetry.CPUPercent < 0 || telemetry.CPUPercent > 100 || telemetry.MemoryUsedBytes > telemetry.MemoryTotalBytes || telemetry.BatteryPercent < 0 || telemetry.BatteryPercent > 100 {
		return HostTelemetry{}, fmt.Errorf("Windows host telemetry is outside valid bounds")
	}
	return telemetry, nil
}

func ParseNVIDIACSV(input io.Reader) ([]NVIDIAGPU, error) {
	scanner := bufio.NewScanner(input)
	var result []NVIDIAGPU
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) != 5 {
			return nil, fmt.Errorf("parse NVIDIA telemetry: expected 5 fields")
		}
		values := make([]int, 4)
		for index := range values {
			value, err := strconv.Atoi(strings.TrimSpace(fields[index+1]))
			if err != nil {
				return nil, fmt.Errorf("parse NVIDIA telemetry: %w", err)
			}
			values[index] = value
		}
		result = append(result, NVIDIAGPU{Name: strings.TrimSpace(fields[0]), UtilizationPercent: values[0], MemoryUsedMiB: values[1], MemoryTotalMiB: values[2], TemperatureC: values[3]})
	}
	return result, scanner.Err()
}

func RenderProviderStatus(status provider.StatusSnapshot, host HostTelemetry, now time.Time) string {
	connection := "disconnected"
	if status.Connected {
		connection = "connected"
	}
	uptime := now.Sub(status.StartedAt).Round(time.Second)
	if uptime < 0 {
		uptime = 0
	}
	power := "battery"
	if host.OnACPower {
		power = "AC"
	}
	var output strings.Builder
	fmt.Fprintf(&output, "Provider %s | uptime %s | requests %d | input %d | output %d | compute %d ms\n", connection, uptime, status.Requests, status.InputTokens, status.OutputTokens, status.ComputeMilliseconds)
	fmt.Fprintf(&output, "Host CPU %.1f%% | RAM %.1f/%.1f GiB | %s %d%%\n", host.CPUPercent, float64(host.MemoryUsedBytes)/(1<<30), float64(host.MemoryTotalBytes)/(1<<30), power, host.BatteryPercent)
	for _, gpu := range host.GPUs {
		fmt.Fprintf(&output, "GPU %s | %d%% | %d/%d MiB | %d C\n", gpu.Name, gpu.UtilizationPercent, gpu.MemoryUsedMiB, gpu.MemoryTotalMiB, gpu.TemperatureC)
	}
	for _, model := range host.LoadedModels {
		fmt.Fprintf(&output, "Loaded %s (%s)\n", model.Name, model.ID)
	}
	for _, offer := range status.Offers {
		health := "healthy"
		if !offer.Healthy {
			health = "unhealthy"
			if offer.Error != "" {
				health += ": " + offer.Error
			}
		}
		fmt.Fprintf(&output, "Offer %s | %s | %s\n", offer.OfferID, offer.Model, health)
	}
	return output.String()
}
