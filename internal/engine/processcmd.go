package engine

import (
	"os/exec"
	"strconv"
	"strings"

	"durablewindows/internal/logger"
)

// CaptureProcessInfo runs PowerShell to capture command lines for all running
// processes and stores them in processCmd so that WindowMetrics.ProcessExePath
// is populated during capture.
func (p *Processor) CaptureProcessInfo() {
	// Use PowerShell on Win10+ for reliable output
	cmd := exec.Command("powershell.exe",
		"-NoProfile", "-Command",
		"Get-CimInstance Win32_Process | Select-Object ProcessId, CommandLine | Format-List",
	)
	output, err := cmd.Output()
	if err != nil {
		// Fall back to WMIC on older Windows
		p.captureProcessInfoWmic()
		return
	}

	var currentPid uint32
	var currentCmdline string
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if currentPid != 0 && currentCmdline != "" {
				p.processCmd[currentPid] = currentCmdline
			}
			currentPid = 0
			currentCmdline = ""
			continue
		}

		if strings.HasPrefix(line, "ProcessId") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if pid, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32); err == nil {
					currentPid = uint32(pid)
				}
			}
		} else if strings.HasPrefix(line, "CommandLine") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentCmdline = strings.TrimSpace(parts[1])
			}
		} else if currentPid != 0 && currentCmdline != "" {
			// Continuation of multi-line command line
			currentCmdline += strings.TrimSpace(line)
		}
	}

	// Flush last entry
	if currentPid != 0 && currentCmdline != "" {
		p.processCmd[currentPid] = currentCmdline
	}

	logger.AutoCapture("", "Captured %d process command lines", len(p.processCmd))
}

func (p *Processor) captureProcessInfoWmic() {
	cmd := exec.Command("wmic.exe", "process", "get", "commandline,processid", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node,") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}
		pid, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			continue
		}
		p.processCmd[uint32(pid)] = fields[1]
	}

	logger.AutoCapture("", "Captured %d process command lines (wmic)", len(p.processCmd))
}
