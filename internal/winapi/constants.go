package winapi

// WinEvent constants
const (
	EVENT_MIN                           = 0x00000001
	EVENT_MAX                           = 0x7FFFFFFF
	EVENT_SYSTEM_FOREGROUND             = 0x0003
	EVENT_SYSTEM_MENUSTART              = 0x0004
	EVENT_SYSTEM_MENUEND                = 0x0005
	EVENT_SYSTEM_MENUPOPUPSTART         = 0x0006
	EVENT_SYSTEM_MENUPOPUPEND           = 0x0007
	EVENT_SYSTEM_CAPTURESTART           = 0x0008
	EVENT_SYSTEM_CAPTUREEND             = 0x0009
	EVENT_SYSTEM_MOVESIZESTART          = 0x000A
	EVENT_SYSTEM_MOVESIZEEND            = 0x000B
	EVENT_SYSTEM_CONTEXTHELPSTART       = 0x000C
	EVENT_SYSTEM_CONTEXTHELPEND         = 0x000D
	EVENT_SYSTEM_DRAGDROPSTART          = 0x000E
	EVENT_SYSTEM_DRAGDROPEND            = 0x000F
	EVENT_SYSTEM_DIALOGSTART            = 0x0010
	EVENT_SYSTEM_DIALOGEND              = 0x0011
	EVENT_SYSTEM_SCROLLINGSTART         = 0x0012
	EVENT_SYSTEM_SCROLLINGEND           = 0x0013
	EVENT_SYSTEM_SWITCHSTART            = 0x0014
	EVENT_SYSTEM_SWITCHEND              = 0x0015
	EVENT_SYSTEM_MINIMIZESTART          = 0x0016
	EVENT_SYSTEM_MINIMIZEEND            = 0x0017
	EVENT_SYSTEM_DESKTOPSWITCH          = 0x0020
	EVENT_SYSTEM_SWITCHER_APPGRABBED    = 0x0024
	EVENT_SYSTEM_SWITCHER_APPOVERTARGET = 0x0025
	EVENT_SYSTEM_SWITCHER_APPDROPPED    = 0x0026
	EVENT_SYSTEM_SWITCHER_CANCELLED     = 0x0027
	EVENT_SYSTEM_IME_KEY_NOTIFICATION   = 0x0029
	EVENT_SYSTEM_END                    = 0x00FF
	EVENT_OBJECT_CREATE                 = 0x8000
	EVENT_OBJECT_DESTROY                = 0x8001
	EVENT_OBJECT_SHOW                   = 0x8002
	EVENT_OBJECT_HIDE                   = 0x8003
	EVENT_OBJECT_REORDER                = 0x8004
	EVENT_OBJECT_LOCATIONCHANGE         = 0x800B
	EVENT_OBJECT_NAMECHANGE             = 0x800C

	WINEVENT_OUTOFCONTEXT   = 0x0000
	WINEVENT_SKIPOWNTHREAD  = 0x0001
	WINEVENT_SKIPOWNPROCESS = 0x0002
	WINEVENT_INCONTEXT      = 0x0004
)

// Mouse event flags
const (
	MOUSEEVENTF_MOVE       = 0x0001
	MOUSEEVENTF_LEFTDOWN   = 0x0002
	MOUSEEVENTF_LEFTUP     = 0x0004
	MOUSEEVENTF_RIGHTDOWN  = 0x0008
	MOUSEEVENTF_RIGHTUP    = 0x0010
	MOUSEEVENTF_MIDDLEDOWN = 0x0020
	MOUSEEVENTF_MIDDLEUP   = 0x0040
	MOUSEEVENTF_WHEEL      = 0x0800
	MOUSEEVENTF_ABSOLUTE   = 0x8000
)

// ShowWindow commands
const (
	SW_HIDE            = 0
	SW_SHOWNORMAL      = 1
	SW_SHOWMINIMIZED   = 2
	SW_SHOWMAXIMIZED   = 3
	SW_SHOWNOACTIVATE  = 4
	SW_SHOW            = 5
	SW_MINIMIZE        = 6
	SW_SHOWMINNOACTIVE = 7
	SW_SHOWNA          = 8
	SW_RESTORE         = 9
	SW_SHOWDEFAULT     = 10
	SW_FORCEMINIMIZE   = 11
)

// SetWindowPos flags
const (
	SWP_NOSIZE         = 0x0001
	SWP_NOMOVE         = 0x0002
	SWP_NOZORDER       = 0x0004
	SWP_NOREDRAW       = 0x0008
	SWP_NOACTIVATE     = 0x0010
	SWP_DRAWFRAME      = 0x0020
	SWP_FRAMECHANGED   = 0x0020
	SWP_SHOWWINDOW     = 0x0040
	SWP_HIDEWINDOW     = 0x0080
	SWP_NOCOPYBITS     = 0x0100
	SWP_NOOWNERZORDER  = 0x0200
	SWP_NOREPOSITION   = 0x0200
	SWP_NOSENDCHANGING = 0x0400
)

// DeferWindowPos flags (same as SWP for most, aliased)
const (
	DWP_DRAWFRAME      = 0x0020
	DWP_FRAMECHANGED   = 0x0020
	DWP_HIDEWINDOW     = 0x0080
	DWP_NOACTIVATE     = 0x0010
	DWP_NOCOPYBITS     = 0x0100
	DWP_NOMOVE         = 0x0002
	DWP_NOOWNERZORDER  = 0x0200
	DWP_NOREDRAW       = 0x0008
	DWP_NOREPOSITION   = 0x0200
	DWP_NOSENDCHANGING = 0x0400
	DWP_NOSIZE         = 0x0001
	DWP_NOZORDER       = 0x0004
	DWP_SHOWWINDOW     = 0x0040
)

// Window style flags
const (
	WS_POPUP            = 0x80000000
	WS_POPUPWINDOW      = 0x80880000
	WS_OVERLAPPED       = 0x00000000
	WS_TILED            = 0x00000000
	WS_TABSTOP          = 0x00010000
	WS_MAXIMIZEBOX      = 0x00010000
	WS_GROUP            = 0x00020000
	WS_MINIMIZEBOX      = 0x00020000
	WS_THICKFRAME       = 0x00040000
	WS_SIZEBOX          = 0x00040000
	WS_SYSMENU          = 0x00080000
	WS_HSCROLL          = 0x00100000
	WS_VSCROLL          = 0x00200000
	WS_DLGFRAME         = 0x00400000
	WS_BORDER           = 0x00800000
	WS_CAPTION          = 0x00C00000
	WS_TILEDWINDOW      = 0x00CF0000
	WS_OVERLAPPEDWINDOW = 0x00CF0000
	WS_MAXIMIZE         = 0x01000000
	WS_CLIPCHILDREN     = 0x02000000
	WS_CLIPSIBLINGS     = 0x04000000
	WS_DISABLED         = 0x08000000
	WS_VISIBLE          = 0x10000000
	WS_MINIMIZE         = 0x20000000
	WS_ICONIC           = 0x20000000
	WS_CHILD            = 0x40000000
	WS_CHILDWINDOW      = 0x40000000
)

// Extended window style constants
const (
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_APPWINDOW   = 0x00040000
	WS_EX_NOACTIVATE  = 0x08000000
)

// GetWindowLong indexes
const (
	GWL_WNDPROC    = -4
	GWL_HINSTANCE  = -6
	GWL_HWNDPARENT = -8
	GWL_STYLE      = -16
	GWL_EXSTYLE    = -20
	GWL_USERDATA   = -21
	GWL_ID         = -12
)

// GetClassLong indexes
const (
	GCLP_HICON   = -14
	GCLP_HICONSM = -34
)

// GetWindow commands
const (
	GW_HWNDFIRST    = 0
	GW_HWNDLAST     = 1
	GW_HWNDNEXT     = 2
	GW_HWNDPREV     = 3
	GW_OWNER        = 4
	GW_CHILD        = 5
	GW_ENABLEDPOPUP = 6
)

// GetAncestor flags
const (
	GA_PARENT    = 1
	GA_ROOT      = 2
	GA_ROOTOWNER = 3
)

// MonitorFromPoint flags
const (
	MONITOR_DEFAULTTONULL    = 0
	MONITOR_DEFAULTTOPRIMARY = 1
	MONITOR_DEFAULTTONEAREST = 2
)

// Window messages
const (
	WM_NULL                        = 0x0000
	WM_CREATE                      = 0x0001
	WM_DESTROY                     = 0x0002
	WM_MOVE                        = 0x0003
	WM_SIZE                        = 0x0005
	WM_ACTIVATE                    = 0x0006
	WM_SETFOCUS                    = 0x0007
	WM_KILLFOCUS                   = 0x0008
	WM_ENABLE                      = 0x000A
	WM_SETREDRAW                   = 0x000B
	WM_SETTEXT                     = 0x000C
	WM_GETTEXT                     = 0x000D
	WM_GETTEXTLENGTH               = 0x000E
	WM_PAINT                       = 0x000F
	WM_CLOSE                       = 0x0010
	WM_QUERYENDSESSION             = 0x0011
	WM_QUIT                        = 0x0012
	WM_QUERYOPEN                   = 0x0013
	WM_ERASEBKGND                  = 0x0014
	WM_SYSCOLORCHANGE              = 0x0015
	WM_ENDSESSION                  = 0x0016
	WM_SHOWWINDOW                  = 0x0018
	WM_SETTINGCHANGE               = 0x001A
	WM_DEVMODECHANGE               = 0x001B
	WM_ACTIVATEAPP                 = 0x001C
	WM_FONTCHANGE                  = 0x001D
	WM_TIMECHANGE                  = 0x001E
	WM_CANCELMODE                  = 0x001F
	WM_SETCURSOR                   = 0x0020
	WM_MOUSEACTIVATE               = 0x0021
	WM_CHILDACTIVATE               = 0x0022
	WM_QUEUESYNC                   = 0x0023
	WM_GETMINMAXINFO               = 0x0024
	WM_PAINTICON                   = 0x0026
	WM_ICONERASEBKGND              = 0x0027
	WM_NEXTDLGCTL                  = 0x0028
	WM_SPOOLERSTATUS               = 0x002A
	WM_DRAWITEM                    = 0x002B
	WM_MEASUREITEM                 = 0x002C
	WM_DELETEITEM                  = 0x002D
	WM_VKEYTOITEM                  = 0x002E
	WM_CHARTOITEM                  = 0x002F
	WM_SETFONT                     = 0x0030
	WM_GETFONT                     = 0x0031
	WM_SETHOTKEY                   = 0x0032
	WM_GETHOTKEY                   = 0x0033
	WM_QUERYDRAGICON               = 0x0037
	WM_COMPAREITEM                 = 0x0039
	WM_GETOBJECT                   = 0x003D
	WM_COMPACTING                  = 0x0041
	WM_COMMNOTIFY                  = 0x0044
	WM_WINDOWPOSCHANGING           = 0x0046
	WM_WINDOWPOSCHANGED            = 0x0047
	WM_POWER                       = 0x0048
	WM_COPYDATA                    = 0x004A
	WM_CANCELJOURNAL               = 0x004B
	WM_NOTIFY                      = 0x004E
	WM_INPUTLANGCHANGEREQUEST      = 0x0050
	WM_INPUTLANGCHANGE             = 0x0051
	WM_TCARD                       = 0x0052
	WM_HELP                        = 0x0053
	WM_USERCHANGED                 = 0x0054
	WM_NOTIFYFORMAT                = 0x0055
	WM_CONTEXTMENU                 = 0x007B
	WM_STYLECHANGING               = 0x007C
	WM_STYLECHANGED                = 0x007D
	WM_DISPLAYCHANGE               = 0x007E
	WM_GETICON                     = 0x007F
	WM_SETICON                     = 0x0080
	WM_NCCREATE                    = 0x0081
	WM_NCDESTROY                   = 0x0082
	WM_NCCALCSIZE                  = 0x0083
	WM_NCHITTEST                   = 0x0084
	WM_NCPAINT                     = 0x0085
	WM_NCACTIVATE                  = 0x0086
	WM_GETDLGCODE                  = 0x0087
	WM_SYNCPAINT                   = 0x0088
	WM_NCMOUSEMOVE                 = 0x00A0
	WM_NCLBUTTONDOWN               = 0x00A1
	WM_NCLBUTTONUP                 = 0x00A2
	WM_NCLBUTTONDBLCLK             = 0x00A3
	WM_NCRBUTTONDOWN               = 0x00A4
	WM_NCRBUTTONUP                 = 0x00A5
	WM_NCRBUTTONDBLCLK             = 0x00A6
	WM_NCMBUTTONDOWN               = 0x00A7
	WM_NCMBUTTONUP                 = 0x00A8
	WM_NCMBUTTONDBLCLK             = 0x00A9
	WM_NCXBUTTONDOWN               = 0x00AB
	WM_NCXBUTTONUP                 = 0x00AC
	WM_NCXBUTTONDBLCLK             = 0x00AD
	WM_INPUT_DEVICE_CHANGE         = 0x00FE
	WM_INPUT                       = 0x00FF
	WM_KEYFIRST                    = 0x0100
	WM_KEYDOWN                     = 0x0100
	WM_KEYUP                       = 0x0101
	WM_CHAR                        = 0x0102
	WM_DEADCHAR                    = 0x0103
	WM_SYSKEYDOWN                  = 0x0104
	WM_SYSKEYUP                    = 0x0105
	WM_SYSCHAR                     = 0x0106
	WM_SYSDEADCHAR                 = 0x0107
	WM_KEYLAST                     = 0x0109
	WM_INITDIALOG                  = 0x0110
	WM_COMMAND                     = 0x0111
	WM_SYSCOMMAND                  = 0x0112
	WM_TIMER                       = 0x0113
	WM_HSCROLL                     = 0x0114
	WM_VSCROLL                     = 0x0115
	WM_INITMENU                    = 0x0116
	WM_INITMENUPOPUP               = 0x0117
	WM_GESTURE                     = 0x0119
	WM_GESTURENOTIFY               = 0x011A
	WM_MENUSELECT                  = 0x011F
	WM_MENUCHAR                    = 0x0120
	WM_ENTERIDLE                   = 0x0121
	WM_MENURBUTTONUP               = 0x0122
	WM_MENUDRAG                    = 0x0123
	WM_MENUGETOBJECT               = 0x0124
	WM_UNINITMENUPOPUP             = 0x0125
	WM_MENUCOMMAND                 = 0x0126
	WM_CHANGEUISTATE               = 0x0127
	WM_UPDATEUISTATE               = 0x0128
	WM_QUERYUISTATE                = 0x0129
	WM_CTLCOLORMSGBOX              = 0x0132
	WM_CTLCOLOREDIT                = 0x0133
	WM_CTLCOLORLISTBOX             = 0x0134
	WM_CTLCOLORBTN                 = 0x0135
	WM_CTLCOLORDLG                 = 0x0136
	WM_CTLCOLORSCROLLBAR           = 0x0137
	WM_CTLCOLORSTATIC              = 0x0138
	WM_MOUSEFIRST                  = 0x0200
	WM_MOUSEMOVE                   = 0x0200
	WM_LBUTTONDOWN                 = 0x0201
	WM_LBUTTONUP                   = 0x0202
	WM_LBUTTONDBLCLK               = 0x0203
	WM_RBUTTONDOWN                 = 0x0204
	WM_RBUTTONUP                   = 0x0205
	WM_RBUTTONDBLCLK               = 0x0206
	WM_MBUTTONDOWN                 = 0x0207
	WM_MBUTTONUP                   = 0x0208
	WM_MBUTTONDBLCLK               = 0x0209
	WM_MOUSEWHEEL                  = 0x020A
	WM_XBUTTONDOWN                 = 0x020B
	WM_XBUTTONUP                   = 0x020C
	WM_XBUTTONDBLCLK               = 0x020D
	WM_MOUSEHWHEEL                 = 0x020E
	WM_PARENTNOTIFY                = 0x0210
	WM_ENTERMENULOOP               = 0x0211
	WM_EXITMENULOOP                = 0x0212
	WM_NEXTMENU                    = 0x0213
	WM_SIZING                      = 0x0214
	WM_CAPTURECHANGED              = 0x0215
	WM_MOVING                      = 0x0216
	WM_POWERBROADCAST              = 0x0218
	WM_DEVICECHANGE                = 0x0219
	WM_ENTERIDLE_DATA              = 0x021A
	WM_MDICREATE                   = 0x0220
	WM_MDIDESTROY                  = 0x0221
	WM_MDIACTIVATE                 = 0x0222
	WM_MDIRESTORE                  = 0x0223
	WM_MDINEXT                     = 0x0224
	WM_MDIMAXIMIZE                 = 0x0225
	WM_MDITILE                     = 0x0226
	WM_MDICASCADE                  = 0x0227
	WM_MDIICONARRANGE              = 0x0228
	WM_MDIGETACTIVE                = 0x0229
	WM_MDISETMENU                  = 0x0230
	WM_ENTERSIZEMOVE               = 0x0231
	WM_EXITSIZEMOVE                = 0x0232
	WM_DROPFILES                   = 0x0233
	WM_MDIREFRESHMENU              = 0x0234
	WM_POINTERDEVICECHANGE         = 0x0238
	WM_POINTERDEVICEINRANGE        = 0x0239
	WM_POINTERDEVICEOUTOFRANGE     = 0x023A
	WM_TOUCH                       = 0x0240
	WM_NCPOINTERUPDATE             = 0x0241
	WM_NCPOINTERDOWN               = 0x0242
	WM_NCPOINTERUP                 = 0x0243
	WM_POINTERUPDATE               = 0x0245
	WM_POINTERDOWN                 = 0x0246
	WM_POINTERUP                   = 0x0247
	WM_POINTERENTER                = 0x0249
	WM_POINTERLEAVE                = 0x024A
	WM_POINTERACTIVATE             = 0x024B
	WM_POINTERCAPTURECHANGED       = 0x024C
	WM_TOUCHHITTESTING             = 0x024D
	WM_POINTERHWHEEL               = 0x024F
	WM_POINTERROUTEDAWAY           = 0x0252
	WM_POINTERROUTEDTO             = 0x0253
	WM_IME_SETCONTEXT              = 0x0281
	WM_IME_NOTIFY                  = 0x0282
	WM_IME_CONTROL                 = 0x0283
	WM_IME_COMPOSITIONFULL         = 0x0284
	WM_IME_SELECT                  = 0x0285
	WM_IME_CHAR                    = 0x0286
	WM_IME_REQUEST                 = 0x0288
	WM_IME_KEYDOWN                 = 0x0290
	WM_IME_KEYUP                   = 0x0291
	WM_NCMOUSEHOVER                = 0x02A0
	WM_MOUSEHOVER                  = 0x02A1
	WM_NCMOUSELEAVE                = 0x02A2
	WM_MOUSELEAVE                  = 0x02A3
	WM_WTSSESSION_CHANGE           = 0x02B1
	WM_TABLET_FIRST                = 0x02C0
	WM_TABLET_LAST                 = 0x02DF
	WM_CUT                         = 0x0300
	WM_COPY                        = 0x0301
	WM_PASTE                       = 0x0302
	WM_CLEAR                       = 0x0303
	WM_UNDO                        = 0x0304
	WM_RENDERFORMAT                = 0x0305
	WM_RENDERALLFORMATS            = 0x0306
	WM_DESTROYCLIPBOARD            = 0x0307
	WM_DRAWCLIPBOARD               = 0x0308
	WM_PAINTCLIPBOARD              = 0x0309
	WM_VSCROLLCLIPBOARD            = 0x030A
	WM_SIZECLIPBOARD               = 0x030B
	WM_ASKCBFORMATNAME             = 0x030C
	WM_CHANGECBCHAIN               = 0x030D
	WM_HSCROLLCLIPBOARD            = 0x030E
	WM_QUERYNEWPALETTE             = 0x030F
	WM_PALETTEISCHANGING           = 0x0310
	WM_PALETTECHANGED              = 0x0311
	WM_HOTKEY                      = 0x0312
	WM_PRINT                       = 0x0317
	WM_PRINTCLIENT                 = 0x0318
	WM_APPCOMMAND                  = 0x0319
	WM_THEMECHANGED                = 0x031A
	WM_CLIPBOARDUPDATE             = 0x031D
	WM_DWMCOMPOSITIONCHANGED       = 0x031E
	WM_DWMNCRENDERINGCHANGED       = 0x031F
	WM_DWMCOLORIZATIONCOLORCHANGED = 0x0320
	WM_DWMWINDOWMAXIMIZEDCHANGE    = 0x0321
	WM_DWMSENDICONICTHUMBNAIL      = 0x0323
	WM_DWMSENDICONICREPRESENTATION = 0x0325
	WM_GETTITLEBARINFOEX           = 0x033F
	WM_HANDHELDFIRST               = 0x0358
	WM_HANDHELDLAST                = 0x035F
	WM_AFXFIRST                    = 0x0360
	WM_AFXLAST                     = 0x037F
	WM_PENWINFIRST                 = 0x0380
	WM_PENWINLAST                  = 0x038F
	WM_APP                         = 0x8000
	WM_USER                        = 0x0400
)

// System Commands
const (
	SC_MINIMIZE            = 0xF020
	SC_TOGGLE_TASKBAR_LOCK = 424
)

// SendMessageTimeout flags
const (
	SMTO_NORMAL             = 0x0000
	SMTO_BLOCK              = 0x0001
	SMTO_ABORTIFHUNG        = 0x0002
	SMTO_NOTIMEOUTIFNOTHUNG = 0x0008
)

// RedrawWindow flags
const (
	RDW_INVALIDATE      = 0x0001
	RDW_INTERNALPAINT   = 0x0002
	RDW_ERASE           = 0x0004
	RDW_VALIDATE        = 0x0008
	RDW_NOINTERNALPAINT = 0x0010
	RDW_NOERASE         = 0x0020
	RDW_NOCHILDREN      = 0x0040
	RDW_ALLCHILDREN     = 0x0080
	RDW_UPDATENOW       = 0x0100
	RDW_ERASENOW        = 0x0200
	RDW_FRAME           = 0x0400
	RDW_NOFRAME         = 0x0800
)

// Window field offsets for GetWindowLong (specific to Windows internals)
const (
	DWMWA_EXTENDED_FRAME_BOUNDS = 9 // RECT — visible bounds excluding DWM shadow
	DWMWA_CLOAKED               = 14

	// DWMWA_CLOAKED return values
	DWM_NOT_CLOAKED       = 0
	DWM_CLOAKED_APP       = 1 // cloaked by the owning application
	DWM_CLOAKED_SHELL     = 2 // cloaked by the shell (virtual desktop, tablet mode)
	DWM_CLOAKED_INHERITED = 4 // cloaked because owner is cloaked
)

// DPI awareness contexts
const (
	DPI_AWARENESS_CONTEXT_UNAWARE              = -1
	DPI_AWARENESS_CONTEXT_SYSTEM_AWARE         = -2
	DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE    = -3
	DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
	DPI_AWARENESS_CONTEXT_UNAWARE_GDISCALED    = -5
)

// Mouse hook types
const (
	WH_MOUSE    = 7
	WH_MOUSE_LL = 14
)

// Hook action codes
const (
	HC_ACTION = 0
)

// Non-client hit test results
const (
	HTMINBUTTON = 8
	HTCAPTION   = 2
	HTSYSMENU   = 3
	HTCLOSE     = 20
)

// Icon constants
const (
	ICON_SMALL  = 0
	ICON_BIG    = 1
	ICON_SMALL2 = 2
)

// IDI constants
const IDI_APPLICATION = 32512

// Shell_NotifyIcon messages
const (
	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002
	NIM_SETVERSION = 0x00000004
)

// NOTIFYICONDATA flags
const (
	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_STATE   = 0x00000008
	NIF_INFO    = 0x00000010
	NIF_GUID    = 0x00000020
)

// NOTIFYICONDATA dwInfoFlags
const (
	NIIF_NONE       = 0x00000000
	NIIF_INFO       = 0x00000001
	NIIF_WARNING    = 0x00000002
	NIIF_ERROR      = 0x00000003
	NIIF_USER       = 0x00000004
	NIIF_LARGE_ICON = 0x00000020
)

// NOTIFYICONDATA dwState / dwStateMask
const (
	NIS_HIDDEN     = 0x00000001
	NIS_SHAREDICON = 0x00000002
)

// Shell_NotifyIcon callback message IDs
const (
	WM_TRAYICON     = WM_APP + 1
	WM_APP_START    = WM_APP + 2
	WM_APP_PARKED   = WM_APP + 3
	WM_APP_SHUTDOWN = WM_APP + 4 // graceful Ctrl+C shutdown
)

// Taskbar messages
const (
	ABM_NEW            = 0x00
	ABM_REMOVE         = 0x01
	ABM_QUERYPOS       = 0x02
	ABM_SETPOS         = 0x03
	ABM_GETSTATE       = 0x04
	ABM_GETTASKBARPOS  = 0x05
	ABM_GETAUTOHIDEBAR = 0x07
	ABM_SETAUTOHIDEBAR = 0x08
	ABM_SETSTATE       = 0x0A
)

// Taskbar edges
const (
	ABE_LEFT   = 0
	ABE_TOP    = 1
	ABE_RIGHT  = 2
	ABE_BOTTOM = 3
)

// Taskbar states
const (
	ABS_AUTOHIDE    = 0x01
	ABS_ALWAYSONTOP = 0x02
)

// Query User Notification State
const (
	QUNS_NOT_PRESENT             = 1
	QUNS_BUSY                    = 2
	QUNS_RUNNING_D3D_FULL_SCREEN = 3
	QUNS_PRESENTATION_MODE       = 4
	QUNS_ACCEPTS_NOTIFICATIONS   = 5
	QUNS_QUIET_TIME              = 6
)

// Session change notification codes (wParam for WM_WTSSESSION_CHANGE)
const (
	WTS_CONSOLE_CONNECT        = 0x1
	WTS_CONSOLE_DISCONNECT     = 0x2
	WTS_REMOTE_CONNECT         = 0x3
	WTS_REMOTE_DISCONNECT      = 0x4
	WTS_SESSION_LOGON          = 0x5
	WTS_SESSION_LOGOFF         = 0x6
	WTS_SESSION_LOCK           = 0x7
	WTS_SESSION_UNLOCK         = 0x8
	WTS_SESSION_REMOTE_CONTROL = 0x9
)

// Power broadcast events (wParam for WM_POWERBROADCAST)
const (
	PBT_APMQUERYSUSPEND       = 0x0000
	PBT_APMQUERYSTANDBY       = 0x0001
	PBT_APMQUERYSUSPENDFAILED = 0x0002
	PBT_APMQUERYSTANDBYFAILED = 0x0003
	PBT_APMSUSPEND            = 0x0004
	PBT_APMSTANDBY            = 0x0005
	PBT_APMRESUMECRITICAL     = 0x0006
	PBT_APMRESUMESUSPEND      = 0x0007
	PBT_APMRESUMESTANDBY      = 0x0008
	PBT_APMBATTERYLOW         = 0x0009
	PBT_APMPOWERSTATUSCHANGE  = 0x000A
	PBT_APMOEMEVENT           = 0x000B
	PBT_APMRESUMEAUTOMATIC    = 0x0012
	PBT_POWERSETTINGCHANGE    = 0x8013
)

// Window placement flags
const (
	WPF_SETMINPOSITION       = 0x0001
	WPF_RESTORETOMAXIMIZED   = 0x0002
	WPF_ASYNCWINDOWPLACEMENT = 0x0004
)

// CURSORINFO flags
const (
	CURSOR_SHOWING = 0x00000001
)

// Virtual-Key Codes
const (
	VK_LBUTTON    = 0x01
	VK_RBUTTON    = 0x02
	VK_CANCEL     = 0x03
	VK_MBUTTON    = 0x04
	VK_BACK       = 0x08
	VK_TAB        = 0x09
	VK_CLEAR      = 0x0C
	VK_RETURN     = 0x0D
	VK_SHIFT      = 0x10
	VK_CONTROL    = 0x11
	VK_MENU       = 0x12
	VK_PAUSE      = 0x13
	VK_CAPITAL    = 0x14
	VK_ESCAPE     = 0x1B
	VK_SPACE      = 0x20
	VK_PRIOR      = 0x21
	VK_NEXT       = 0x22
	VK_END        = 0x23
	VK_HOME       = 0x24
	VK_LEFT       = 0x25
	VK_UP         = 0x26
	VK_RIGHT      = 0x27
	VK_DOWN       = 0x28
	VK_SELECT     = 0x29
	VK_PRINT      = 0x2A
	VK_EXECUTE    = 0x2B
	VK_SNAPSHOT   = 0x2C
	VK_INSERT     = 0x2D
	VK_DELETE     = 0x2E
	VK_HELP       = 0x2F
	VK_0          = 0x30
	VK_1          = 0x31
	VK_2          = 0x32
	VK_3          = 0x33
	VK_4          = 0x34
	VK_5          = 0x35
	VK_6          = 0x36
	VK_7          = 0x37
	VK_8          = 0x38
	VK_9          = 0x39
	VK_A          = 0x41
	VK_B          = 0x42
	VK_C          = 0x43
	VK_D          = 0x44
	VK_E          = 0x45
	VK_F          = 0x46
	VK_G          = 0x47
	VK_H          = 0x48
	VK_I          = 0x49
	VK_J          = 0x4A
	VK_K          = 0x4B
	VK_L          = 0x4C
	VK_M          = 0x4D
	VK_N          = 0x4E
	VK_O          = 0x4F
	VK_P          = 0x50
	VK_Q          = 0x51
	VK_R          = 0x52
	VK_S          = 0x53
	VK_T          = 0x54
	VK_U          = 0x55
	VK_V          = 0x56
	VK_W          = 0x57
	VK_X          = 0x58
	VK_Y          = 0x59
	VK_Z          = 0x5A
	VK_LWIN       = 0x5B
	VK_RWIN       = 0x5C
	VK_APPS       = 0x5D
	VK_SLEEP      = 0x5F
	VK_NUMPAD0    = 0x60
	VK_NUMPAD1    = 0x61
	VK_NUMPAD2    = 0x62
	VK_NUMPAD3    = 0x63
	VK_NUMPAD4    = 0x64
	VK_NUMPAD5    = 0x65
	VK_NUMPAD6    = 0x66
	VK_NUMPAD7    = 0x67
	VK_NUMPAD8    = 0x68
	VK_NUMPAD9    = 0x69
	VK_F1         = 0x70
	VK_F2         = 0x71
	VK_F3         = 0x72
	VK_F4         = 0x73
	VK_F5         = 0x74
	VK_F6         = 0x75
	VK_F7         = 0x76
	VK_F8         = 0x77
	VK_F9         = 0x78
	VK_F10        = 0x79
	VK_F11        = 0x7A
	VK_F12        = 0x7B
	VK_OEM_3      = 0xC0
	VK_OEM_MINUS  = 0xBD
	VK_OEM_PLUS   = 0xBB
	VK_OEM_PERIOD = 0xBE
)

// Extended window styles (additional)
const (
	WS_EX_DLGMODALFRAME = 0x00000001
	WS_EX_CLIENTEDGE    = 0x00000200
)

// Button styles
const (
	BS_PUSHBUTTON    = 0x00000000
	BS_DEFPUSHBUTTON = 0x00000001
)

// System colors
const (
	COLOR_BTNFACE = 15
)

// Key modifier flags for RegisterHotKey
const (
	MOD_ALT      = 0x0001
	MOD_CONTROL  = 0x0002
	MOD_SHIFT    = 0x0004
	MOD_WIN      = 0x0008
	MOD_NOREPEAT = 0x4000
)
