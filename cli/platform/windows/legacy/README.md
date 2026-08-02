# Myference Windows CLI (legacy implementation)

Myference turns this Windows laptop into an on-demand Ollama inference server for a trusted private home network. It includes a PowerShell lifecycle manager and a native Go streaming gateway/dashboard.

## What it does

- Runs Ollama privately on `127.0.0.1:11435`.
- Exposes the tracked gateway on `0.0.0.0:11434` for Ollama and OpenAI-compatible clients.
- Restricts its managed Windows Firewall rule to Private networks and `LocalSubnet`.
- Shows live CPU, RAM, AC/battery status, charge and remaining time, NVIDIA GPU/VRAM/temperature/power, loaded models, endpoints, uptime, request totals, active requests, client IPs, routes, duration, status, and recent traffic.
- Preserves streaming responses; inference tokens are forwarded as Ollama produces them.
- Keeps the machine awake, assigns Ollama a safe high priority, and can temporarily stop configured optional apps/services.
- Records every reversible change and restores the normal Windows workspace when stopped.

Ollama has no built-in HTTP authentication. Anyone permitted by the firewall on the trusted subnet can submit inference requests. Never forward port 11434 on the router or use Myference on a Public network.

## Current-terminal mode

Open PowerShell as Administrator in this directory and quit the normal Ollama tray application first:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\myference.ps1 doctor
.\myference.ps1 start
```

`start` opens the full live dashboard in the same terminal. Press `Q` to stop the gateway and Ollama and restore all recorded Windows state.

Optional focus modes:

```powershell
.\myference.ps1 start -Focus
.\myference.ps1 start -Exclusive
```

`-Focus` stops the configured optional apps/services. `-Exclusive` also reduces the current desktop session, but Windows Home can relaunch Explorer; use headless mode for a genuine no-Explorer session.

For an unattended/background start, use `-NoDashboard`. Attach a UI later with:

```powershell
.\myference.ps1 dashboard
```

When attached this way, `Q` only closes the viewer; the separately running server remains active. Stop it with `myference.ps1 stop`.

## Headless login mode (Windows Home compatible)

Save all work and run from Administrator PowerShell:

```powershell
.\myference.ps1 headless
```

Press Enter to sign out immediately, Esc to cancel and roll back, or wait for the 10-second automatic sign-out. Sign in again. The Myference dashboard becomes the login shell, Explorer is not started, and the elevated server task starts automatically.

After the headless server is successfully ready, Myference records the current lid actions and temporarily changes closing the lid to **Do nothing** on both AC and battery. Press `Q` in the dashboard to stop the server, restore both original lid actions and the original shell policy, and sign out. The next login returns to the full normal desktop. `headless-restore` performs the same server, lid, power, and shell recovery in an emergency. Use `headless-install` instead if you want to prepare the next login without signing out immediately.

Emergency recovery from a blank headless screen:

1. Press Ctrl+Shift+Esc and select **Run new task**.
2. Enable **Create this task with administrative privileges**.
3. Run:

```powershell
powershell.exe -ExecutionPolicy Bypass -File K:\myference\myference.ps1 headless-restore
```

Then sign out and back in.

## Use it from another laptop

The dashboard prints the current LAN URL. Native Ollama API examples:

```powershell
curl.exe http://SERVER_IP:11434/api/tags
curl.exe http://SERVER_IP:11434/api/generate -H "Content-Type: application/json" -d "{\"model\":\"YOUR_MODEL\",\"prompt\":\"Hello\",\"stream\":false}"
```

For OpenAI-compatible clients:

```text
Base URL: http://SERVER_IP:11434/v1
API key:  ollama
```

The placeholder key only satisfies clients that require a value; it is not authentication.

## Commands

```text
myference.ps1 doctor
myference.ps1 status
myference.ps1 start [-Port 11434] [-Focus | -Exclusive] [-NoFirewall]
myference.ps1 dashboard
myference.ps1 stop
myference.ps1 focus [-Apply]
myference.ps1 restore
myference.ps1 models
myference.ps1 test [-Model installed-name]
myference.ps1 lan-check
myference.ps1 headless
myference.ps1 headless-install
myference.ps1 headless-status
myference.ps1 headless-restore
```

`focus` without `-Apply` is a preview. `restore` is an emergency manual rollback; normal dashboard sessions only need `Q`.

## Configuration

Copy `myference.config.example.json` to `myference.config.json` and customize it. Important fields:

```json
{
  "port": 11434,
  "backendPort": 11435,
  "allowedRemoteAddress": "LocalSubnet",
  "keepAlive": "-1",
  "preloadModel": "qwen3.5:9b",
  "maxLoadedModels": 1,
  "numParallel": 1,
  "contextLength": 4096,
  "flashAttention": true,
  "kvCacheType": "q8_0",
  "performancePowerPlan": true,
  "processPriority": "High",
  "requireACPower": true,
  "stopServices": [],
  "stopProcesses": ["OneDrive", "ms-teams", "Teams", "Discord", "Spotify"]
}
```

`port` is the LAN gateway. `backendPort` is loopback-only and must differ. When `preloadModel` is set, Myference loads that installed model before announcing readiness; `keepAlive: "-1"` then keeps it resident until stop, moving cold-start latency to server startup. Myference defaults to a 4K context, Flash Attention, and the recommended q8 KV cache to control memory usage. It switches to the High Performance power plan for the server session and restores the previous plan on stop. For tighter access, replace `LocalSubnet` with a CIDR such as `192.168.0.0/24`.

## Model sizing and performance

For fast inference, the loaded model plus context cache should fit almost entirely in dedicated VRAM. Check the dashboard model line or `ollama ps`: `100% GPU` is the target. A model larger than VRAM is split into system RAM and CPU work; near-full RAM then causes paging and sharply lower token speed even though Task Manager still shows some GPU activity.

On an 8 GB RTX 4060, prefer roughly 3-6 GB Q4 models. `qwen3:8b` (5.2 GB) is a safe text model; `qwen3.5:4b` (3.4 GB) is a newer multimodal option. `qwen3.5:9b` (6.6 GB weights) may fit only with a short context and other GPU applications closed, so verify that `ollama ps` reports full GPU placement.

A model's advertised maximum context is not a target allocation. Larger contexts consume more VRAM/RAM and reduce performance. Increase `contextLength` only when the workload truly needs it.
Only add optional services after verifying the laptop does not need them for networking, GPU drivers, authentication, storage, security, or power management. Myference deliberately does not disable Defender, core networking, GPU services, Windows Update components permanently, drivers, or optional Windows features.

## Dashboard build

The Windows executable is included. To rebuild it after changing `dashboard.go`, install Go and run:

```powershell
go build -o myference-dashboard.exe dashboard.go
```

It uses only the Go standard library. Request telemetry is produced by the streaming reverse proxy because Ollama itself does not expose per-request activity. The private `/_myference/metrics` endpoint accepts loopback clients only.

## Other behavior

The keep-awake worker prevents automatic idle sleep but does not override closing the lid or manually selecting Sleep. Applications are reopened from recorded executable paths, while their exact tabs/documents depend on each application's own restore behavior. Save active work before Focus, Exclusive, or headless mode.

`install-startup.ps1` can register an elevated sign-in task for background service mode:

```powershell
.\install-startup.ps1 -Focus
.\install-startup.ps1 -Remove
```

Use `myference.ps1 dashboard` to view an instance started by that task.
