$ErrorActionPreference = 'Stop'
Add-Type @"
using System;
using System.Runtime.InteropServices;
public static class MyferencePower {
    [DllImport("kernel32.dll")]
    public static extern uint SetThreadExecutionState(uint esFlags);
}
"@
$ES_CONTINUOUS = 0x80000000
$ES_SYSTEM_REQUIRED = 0x00000001
[void][MyferencePower]::SetThreadExecutionState($ES_CONTINUOUS -bor $ES_SYSTEM_REQUIRED)
try {
    while ($true) { Start-Sleep -Seconds 30 }
} finally {
    [void][MyferencePower]::SetThreadExecutionState($ES_CONTINUOUS)
}
