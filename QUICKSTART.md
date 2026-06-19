
## DurableWindows Quick Help

A Windows system-tray utility that remembers window positions and restores them
when display configurations change (monitor connect/disconnect, RDP, sleep/resume).

### Tray Icon

| Action | Result |
| --- | --- |
| Double-click icon | Capture snapshot 0 (save all window positions) |
| Single-click icon | Restore snapshot 0 (windows snap back to saved positions) |
| Right-click icon | Context menu with capture, restore, pause, and parked window options |

All window positions are automatically persisted to disk on every capture cycle. Snapshots survive application restarts.

### Shift+Minimize to Tray

Hold **Shift** while minimizing any window to park it to the system
tray. A tray icon appears for the parked window — click it to restore
the window to its previous position.

| Action | Result |
| --- | --- |
| Shift + minimize button | Park window to tray (individual icon appears) |
| Shift + Win+Down | Same — park to tray |
| Click parked window's tray icon | Restore that window to its previous position |
| Tray menu → Restore all parked windows | Restore every parked window at once |

To disable the feature:

```
-disable_window_parking
```

Parked windows are remembered across capture cycles and appear in the
tray menu's Restore submenu.

### Command Line Options

| Flag | Default | Description |
| --- | --- | --- |
| `-silent` | false | Silent mode (no balloon tips, no console output) |
| `-log` | (snapshot only) | Log categories: `filtered_events`, `automatic_capture_restore`, `snapshot_capture_restore`, `window_events`, `tray_interaction`, `window_parking`. Comma-separated, e.g. `-log snapshot_capture_restore,tray_interaction` |
| `-nolog` | (none) | Invert specific categories after `-log`. Applied second, so a category in both ends up disabled. Useful with `-log all`, e.g. `-log all -nolog filtered_events` |
| `-portable_mode` | false | Store data files under `user_data/` within the program folder |
| `-redirect_appdata <path>` | (AppData) | Override the app data directory. Use `.` for current directory |
| `-delay_auto_capture <seconds>` | 3.0 | Lag between window move event and automatic capture |
| `-delay_auto_restore <seconds>` | 0.5 | Lag between display change and automatic restore |
| `-halt_restore <seconds>` | 3 | Seconds to wait before retrying a restore that was interrupted by another display change |
| `-fix_zorder <0\|1\|2>` | 1 | Z-order preservation: 0=none, 1=snapshot restore only, 2=all restores |
| `-disable_offscreen_fix` | false | Disable automatic correction of off-screen windows |
| `-enhanced_offscreen_fix` | false | Use 4-corner off-screen detection instead of 2-corner |
| `-fix_minimized_restore` | true | Handle the positioning of minimized window restores after a display change in place of Windows |
| `-disable_fast_restore` | false | Use slower but more thorough 5-pass restore instead of 2-pass |
| `-disable_window_parking` | false | Disable Shift+minimize-to-tray entirely |
| `-redraw_desktop` | false | Force full desktop redraw after restore |
| `-show_desktop_when_display_changes` | false | Minimize all windows before restoring |
| `-disable_notifications` | false | Disable notification balloons during snapshots and auto-restore |
| `-prompt_session_restore` | false | Ask before restoring windows when unlocking a session |
| `-ignore_process <names>` | (none) | Semicolon-separated list of process names to exclude |
| `-care_process <names>` | (none) | Semicolon-separated list of process names to include (whitelist) |
| `-debug_process <names>` | (none) | Semicolon-separated list of process names to debug |

### One-shot Commands (no tray icon, exits after completion)

| Flag | Description |
| --- | --- |
| `-capture_snapshot <id>` | Capture snapshot by ID (0-37) and exit |
| `-restore_snapshot <id>` | Restore snapshot by ID (0-37) and exit |
| `-restore_parked_windows` | Restore any windows that were left parked (e.g. after a crash) and exit |

### Right-click Menu

| Menu Item | Shortcut | Description |
| --- | --- | --- |
| Capture snapshot | Alt+C | Save positions under a key (0-9, a-z, A-Z, `` ` ``) |
| Restore snapshot | Alt+R | Restore positions from a key (0-9, a-z, A-Z, `` ` ``) |
| Pause auto restore | Alt+E | Temporarily disable automatic display-change restores |
| Restore all parked windows | Alt+P | Restore all windows that were Shift+minimized to the tray |
| Exit | Alt+X | Quit the application |

Parked windows are windows hidden to the system tray via Shift+minimize
(see **Shift+Minimize to Tray** above). Each parked window also gets its
own individual tray icon — click it to restore that specific window.

### Logging Categories

Use `-log <categories>` to control console output. Only `snapshot_capture_restore`
is enabled by default.

| Category | What it shows |
| --- | --- |
| `filtered_events` | Windows examined but skipped (wrong class, cloaked, ghost frames, etc.) |
| `automatic_capture_restore` | Capture cycle stats, lock/sleep/resume restore lifecycle |
| `snapshot_capture_restore` | Manual snapshot capture and restore, per-window restore log |
| `window_events` | Window create/show/hide/kill, foreground change, move/resize, minimize |
| `tray_interaction` | Startup sequence, tray clicks, timers, menu commands |
| `window_parking` | Minimize-to-tray: Shift+minimize interception, park/unpark, parked icon clicks |

**Examples:**
```
# Enable all categories:
-log all

# See everything (explicit):
-log filtered_events,automatic_capture_restore,snapshot_capture_restore,window_events,tray_interaction,window_parking

# See only manual actions and what's being filtered:
-log snapshot_capture_restore,filtered_events

# See only window lifecycle events:
-log window_events

# Enable everything except the noisy filtered_events:
-log all -nolog filtered_events

# Enable everything except filtered_events and tray_interaction:
-log all -nolog filtered_events,tray_interaction
```

### Files

| Location | Content |
| --- | --- |
| `%LocalAppData%\DurableWindows\` | Database and lock files |
| `user_data\` (portable mode) | Database and lock files |
