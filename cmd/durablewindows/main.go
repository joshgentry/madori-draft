package main

import (
	"flag"
	"os"
	"path/filepath"

	"durablewindows/internal/engine"
	"durablewindows/internal/logger"
	"durablewindows/internal/storage"
	"durablewindows/internal/tray"
	"durablewindows/internal/winapi"
)

var (
	version     = "1.0.0"
	productName = "DurableWindows"
)

// CLI flags
var (
	silent               bool
	logCategories        string
	noLogCategories      string
	delayAutoCapture     float64
	delayAutoRestore     float64
	ignoreProcess        string
	careProcess          string
	debugProcess         string
	haltRestore          float64
	fixZorder            int
	showDesktop          bool
	offscreenFix         bool
	enhancedOffscreenFix bool
	fixUnminimizedRestore bool
	restoreSnapshot      int
	captureSnapshot      int
	restoreParkedWindows bool
	portableMode         bool
	redirectAppdata      string
	promptSessionRestore bool
	posMatchThreshold    int
	fixZorderSpecified   bool
	disableNotifications bool
	disableFastRestore   bool
	windowParkingMethod  string
	disableWindowParking bool
	dpiSensitiveCall     bool
	redrawDesktop        bool
)

func main() {
	parseFlags()

	// Initialize logger
	if err := logger.Init(productName); err != nil {
		_ = err // non-fatal
	}
	defer logger.Close()

	logger.SetSilent(silent)
	logger.SetCategories(logCategories)
	logger.DisableCategories(noLogCategories)
	logger.Event("", "%s %s starting", productName, version)

	// Determine app data folder
	appDataFolder := resolveAppDataFolder()
	if err := os.MkdirAll(appDataFolder, 0700); err != nil {
		fatal("Failed to create app data folder: %v", err)
	}

	// Initialize storage (needed for both one-shot and GUI modes)
	logger.Event("", "Opening database...")
	store, err := storage.NewStore(appDataFolder, productName, version)
	if err != nil {
		fatal("Failed to open database: %v", err)
	}
	defer store.Close()
	engine.SetStore(store)
	logger.Event("", "Database opened successfully")

	// Handle one-shot CLI commands (no GUI needed)
	if handleOneShotCommands(appDataFolder) {
		return
	}

	// Create the core engine
	proc := engine.NewProcessor()
	proc.AppDataFolder = appDataFolder

	// Apply CLI settings to processor
	applySettings(proc)

	// Create system tray app
	trayApp := tray.NewTrayApp(proc)
	trayApp.SetGlobalTrayApp(trayApp)
	trayApp.SetNotification(!disableNotifications)

	// Run the message pump (blocks until quit)
	if err := trayApp.Run(); err != nil {
		fatal("Application error: %v", err)
	}

	logger.Event("", "Application exiting normally")
}

// fatal logs an error, flushes stdout, and exits with code 1.
// os.Exit does not flush stdout, so without the Sync the error
// message would be lost when the process terminates.
func fatal(format string, args ...interface{}) {
	logger.Error("", format, args...)
	os.Stdout.Sync()
	os.Exit(1)
}

func parseFlags() {
	flag.BoolVar(&silent, "silent", false, "Silent mode (no balloon tips)")
	flag.StringVar(&logCategories, "log", "", "Log categories: filtered_events,automatic_capture_restore,snapshot_capture_restore,window_events,tray_interaction,window_parking")
	flag.StringVar(&noLogCategories, "nolog", "", "Invert specific log categories (applied after -log, e.g. \"-log all -nolog filtered_events\")")
	flag.Float64Var(&delayAutoCapture, "delay_auto_capture", 0, "Auto capture delay in seconds")
	flag.Float64Var(&delayAutoRestore, "delay_auto_restore", 0, "Auto restore delay in seconds")
	flag.StringVar(&ignoreProcess, "ignore_process", "", "Process names to ignore (semicolon-separated)")
	flag.StringVar(&careProcess, "care_process", "", "Process names to care about (whitelist)")
	flag.StringVar(&debugProcess, "debug_process", "", "Process names to debug")
	flag.Float64Var(&haltRestore, "halt_restore", 3, "Seconds to wait before retrying restore")
	flag.IntVar(&fixZorder, "fix_zorder", 1, "Z-order fix: 0=none, 1=snapshot only, 2=all")
	flag.BoolVar(&showDesktop, "show_desktop_when_display_changes", false, "Show desktop on display change")
	flag.BoolVar(&offscreenFix, "disable_offscreen_fix", false, "Disable off-screen window fix")
	flag.BoolVar(&enhancedOffscreenFix, "enhanced_offscreen_fix", false, "Enhanced off-screen fix")
	flag.BoolVar(&fixUnminimizedRestore, "fix_unminimized_restore", true, "Restore unminimized window position after display change")
	flag.IntVar(&restoreSnapshot, "restore_snapshot", -1, "Restore snapshot by ID")
	flag.IntVar(&captureSnapshot, "capture_snapshot", -1, "Capture snapshot by ID")
	flag.BoolVar(&restoreParkedWindows, "restore_parked_windows", false, "Restore all parked windows from database and exit")
	flag.BoolVar(&portableMode, "portable_mode", false, "Portable mode (data in user_data/)")
	flag.StringVar(&redirectAppdata, "redirect_appdata", "", "Override app data directory")
	flag.BoolVar(&promptSessionRestore, "prompt_session_restore", false, "Prompt before session restore")
	flag.IntVar(&posMatchThreshold, "pos_match_threshold", 40, "Window position match threshold")
	flag.BoolVar(&disableFastRestore, "disable_fast_restore", false, "Disable fast restore")
	flag.BoolVar(&dpiSensitiveCall, "dpi_sensitive_call", false, "Enable DPI-aware thread context switching")
	flag.BoolVar(&redrawDesktop, "redraw_desktop", false, "Force desktop redraw after restore")
	flag.BoolVar(&disableNotifications, "disable_notifications", false, "Disable notification balloons")
	flag.StringVar(&windowParkingMethod, "window_parking_method", "hook", "Window parking method: hook (default) or poll")
	flag.BoolVar(&disableWindowParking, "disable_window_parking", false, "Disable minimize-to-tray entirely")

	flag.Parse()

	// Track whether fix_zorder was explicitly set
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "fix_zorder" {
			fixZorderSpecified = true
		}
	})
}

func resolveAppDataFolder() string {
	if redirectAppdata == "." {
		return "."
	}
	if redirectAppdata == "user_data" || portableMode {
		return "user_data"
	}
	if redirectAppdata != "" {
		return redirectAppdata
	}

	localAppData, err := os.UserCacheDir()
	if err != nil {
		localAppData = "."
	}
	return filepath.Join(localAppData, productName)
}

func handleOneShotCommands(appDataFolder string) bool {
	// Match the DPI-awareness set by the GUI path (tray.go). Without this,
	// GetWindowRect / SetWindowPlacement coordinates are virtualized by
	// Windows and come out wrong on mixed-DPI multi-monitor systems.
	winapi.SetProcessDpiAwarenessContext(winapi.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	proc := engine.NewProcessor()
	proc.AppDataFolder = appDataFolder

	if restoreSnapshot >= 0 {
		proc.RestoreSnapshotCmd(restoreSnapshot)
		return true
	}
	if captureSnapshot >= 0 {
		proc.CaptureSnapshotCmd(captureSnapshot)
		return true
	}
	if restoreParkedWindows {
		proc.RestoreParkedWindowsCmd()
		return true
	}
	return false
}

func applySettings(proc *engine.Processor) {
	proc.UserForcedCaptureLatency = int(delayAutoCapture * 1000)
	proc.UserForcedRestoreLatency = int(delayAutoRestore * 1000)
	proc.MaxDiffPos = posMatchThreshold

	if ignoreProcess != "" {
		proc.SetIgnoreProcess(ignoreProcess)
	}
	if careProcess != "" {
		proc.SetCareProcess(careProcess)
	}
	if debugProcess != "" {
		proc.SetDebugProcess(debugProcess)
	}
	if haltRestore > 0 {
		proc.HaltRestore = int(haltRestore * 1000)
	}

	if fixZorderSpecified {
		proc.FixZorder = fixZorder
	}
	proc.ShowDesktop = showDesktop
	proc.EnableOffScreenFix = !offscreenFix
	proc.EnhancedOffScreenFix = enhancedOffscreenFix
	proc.FixUnminimizedRestore = fixUnminimizedRestore
	proc.PromptSessionRestore = promptSessionRestore
	proc.UsePollParking = (windowParkingMethod == "poll")
	if disableWindowParking {
		proc.EnableMinimizeToTray = false
	}
	proc.RedrawDesktop = redrawDesktop
	proc.DpiSensitiveCall = dpiSensitiveCall
	if disableFastRestore {
		proc.FastRestore = false
	}
}
