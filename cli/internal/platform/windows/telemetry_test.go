package windows

import (
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/provider"
)

func TestParseHostTelemetryAndNVIDIA(t *testing.T) {
	host, err := ParseHostTelemetry(strings.NewReader(`{"cpuPercent":12.5,"memoryUsedBytes":8589934592,"memoryTotalBytes":17179869184,"onACPower":true,"batteryPercent":88}`))
	if err != nil {
		t.Fatal(err)
	}
	if host.CPUPercent != 12.5 || host.MemoryUsedBytes != 8589934592 || host.MemoryTotalBytes != 17179869184 || !host.OnACPower || host.BatteryPercent != 88 {
		t.Fatalf("host=%+v", host)
	}
	gpus, err := ParseNVIDIACSV(strings.NewReader("NVIDIA GeForce RTX 4090, 21, 1024, 24564, 32\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 1 || gpus[0].Name != "NVIDIA GeForce RTX 4090" || gpus[0].UtilizationPercent != 21 || gpus[0].MemoryUsedMiB != 1024 || gpus[0].MemoryTotalMiB != 24564 || gpus[0].TemperatureC != 32 {
		t.Fatalf("gpus=%+v", gpus)
	}
}

func TestRenderProviderStatusIncludesEveryOfferAndHostMetric(t *testing.T) {
	status := provider.StatusSnapshot{Connected: true, StartedAt: time.Now().Add(-time.Minute), Requests: 3, InputTokens: 10, OutputTokens: 20, ComputeMilliseconds: 900, Offers: []provider.OfferStatus{
		{OfferID: "local-a", Model: "qwen", Healthy: true},
		{OfferID: "local-b", Model: "llama", Healthy: false, Error: "preload failed"},
	}}
	host := HostTelemetry{CPUPercent: 12.5, MemoryUsedBytes: 8 << 30, MemoryTotalBytes: 16 << 30, OnACPower: true, BatteryPercent: 88, LoadedModels: []LoadedModel{{Name: "qwen", ID: "abc"}}, GPUs: []NVIDIAGPU{{Name: "RTX", UtilizationPercent: 21}}}
	rendered := RenderProviderStatus(status, host, time.Now())
	for _, expected := range []string{"connected", "requests 3", "input 10", "output 20", "CPU 12.5%", "RAM 8.0/16.0 GiB", "AC 88%", "local-a", "qwen", "healthy", "local-b", "llama", "preload failed", "RTX", "21%"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("missing %q in:\n%s", expected, rendered)
		}
	}
}
