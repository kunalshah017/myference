//go:build windows

package windows

import (
	"bytes"
	"context"
	"os/exec"
)

const hostTelemetryPowerShell = `$cpu=(Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average; $os=Get-CimInstance Win32_OperatingSystem; $battery=Get-CimInstance Win32_Battery | Select-Object -First 1; [pscustomobject]@{cpuPercent=[double]$cpu;memoryUsedBytes=[uint64](($os.TotalVisibleMemorySize-$os.FreePhysicalMemory)*1024);memoryTotalBytes=[uint64]($os.TotalVisibleMemorySize*1024);onACPower=[bool](!$battery -or $battery.BatteryStatus -in 2,6,7,8,9,11);batteryPercent=[int]($(if($battery){$battery.EstimatedChargeRemaining}else{100}))}|ConvertTo-Json -Compress`

func CollectHostTelemetry(ctx context.Context) (HostTelemetry, error) {
	output, err := runWindowsCommand(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", hostTelemetryPowerShell)
	if err != nil {
		return HostTelemetry{}, err
	}
	telemetry, err := ParseHostTelemetry(bytes.NewReader(output))
	if err != nil {
		return HostTelemetry{}, err
	}
	if path, lookErr := exec.LookPath("nvidia-smi.exe"); lookErr == nil {
		if gpuOutput, queryErr := runWindowsCommand(ctx, path, "--query-gpu=name,utilization.gpu,memory.used,memory.total,temperature.gpu", "--format=csv,noheader,nounits"); queryErr == nil {
			telemetry.GPUs, _ = ParseNVIDIACSV(bytes.NewReader(gpuOutput))
		}
	}
	if path, lookErr := exec.LookPath("ollama.exe"); lookErr == nil {
		if modelOutput, queryErr := runWindowsCommand(ctx, path, "ps"); queryErr == nil {
			telemetry.LoadedModels, _ = ParseOllamaPS(bytes.NewReader(modelOutput))
		}
	}
	return telemetry, nil
}
