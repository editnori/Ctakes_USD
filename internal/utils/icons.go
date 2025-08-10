package utils

import (
	"os"
	"runtime"
	"strings"
)

// Terminal capability detection
var (
	supportsEmoji bool
	initialized   bool
)

// Icon sets for terminals
var (
	// Emoji icons for capable terminals - Consistent blue theme with circle emojis
	emojiIcons = map[string]string{
		// Folders - use blue circles for consistency
		"folder":      "🔵", // Blue circle for folders
		"folder_open": "🔵", // Blue circle for open folders

		// Files - white circles for generic files, colored for specific types
		"file":         "⚪", // White circle - generic files
		"file_code":    "🟢", // Green circle - code
		"file_text":    "⚪", // White circle - text
		"file_config":  "🟠", // Orange circle - config
		"file_archive": "🟤", // Brown circle - archives
		"file_image":   "🟣", // Purple circle - images
		"file_audio":   "🔴", // Red circle - audio
		"file_video":   "🟠", // Orange circle - video
		"file_data":    "🔵", // Blue circle - data files
		"file_binary":  "⚫", // Black circle - binaries
		"file_medical": "🟢", // Green circle - medical files

		// Navigation Menu - specific navigation icons
		"monitor":    "📊", // Chart for system monitor
		"browser":    "🔵", // Blue circle for file browser
		"dictionary": "📚", // Books for dictionary builder

		// Status - Colored circles with meaning
		"success": "🟢", // Green = good/success
		"error":   "🔴", // Red = error/problem
		"warning": "🟡", // Yellow = warning/attention
		"info":    "🔵", // Blue = information
		"loading": "🟡", // Yellow = in progress
		"ready":   "🟢", // Green = ready
		"stopped": "⚫", // Black = stopped/inactive
		"paused":  "🟡", // Yellow = paused
		"running": "🟢", // Green = running

		// Actions - Colored circles by type
		"add":      "🟢",  // Green = positive action
		"remove":   "🔴",  // Red = destructive action
		"edit":     "🟡",  // Yellow = modify action
		"save":     "🟢",  // Green = commit action
		"cancel":   "🔴",  // Red = abort action
		"search":   "🔍",  // Magnifying glass = search
		"settings": "⚙️", // Gear = settings
		"refresh":  "🟡",  // Yellow = reload
		"help":     "🔵",  // Blue = information

		// System Resources - Colored circles
		"cpu":      "🔵", // Blue = processing
		"memory":   "🟣", // Purple = memory/storage
		"disk":     "🟤", // Brown = physical storage
		"network":  "🟢", // Green = connected
		"process":  "🟡", // Yellow = running process
		"database": "🟠", // Orange = structured data

		// Medical/Clinical - Consistent icons
		"clinical": "🟢", // Green for health/medical
		"medical":  "🟢", // Green for health/medical
		"pipeline": "🔧", // Wrench for pipeline processing
		"analyze":  "📊", // Chart for analysis
		"document": "⚪", // White circle for documents

		// Dictionary Builder States - All circles
		"dict_main":     "🟣", // Purple (main dictionary)
		"dict_create":   "🟢", // Green (create new)
		"dict_load":     "🔵", // Blue (load existing)
		"dict_config":   "🟠", // Orange (configuration)
		"dict_tui":      "🟣", // Purple (semantic types)
		"dict_vocab":    "🔵", // Blue (vocabularies)
		"dict_build":    "🟢", // Green (build/process)
		"dict_template": "🟠", // Orange (templates/presets)
		"dict_memory":   "🟣", // Purple (memory settings)
		"dict_process":  "🟡", // Yellow (processing options)
		"dict_filter":   "🟡", // Yellow (filter settings)
		"dict_output":   "🟢", // Green (output formats)
		"dict_relation": "🔵", // Blue (relationships)
		"dict_cased":    "🟣", // Purple (cased dictionary)

		// Arrows/Navigation - Keep simple
		"arrow_up":    "▲", // Up triangle
		"arrow_down":  "▼", // Down triangle
		"arrow_left":  "◀", // Left triangle
		"arrow_right": "▶", // Right triangle
		"home":        "🔵", // Blue circle for home
		"back":        "◀", // Left triangle for back
		"forward":     "▶", // Right triangle for forward
	}

	// Unicode box-drawing icons (for terminals without emoji but with Unicode)
	unicodeIcons = map[string]string{
		"folder":       "▤",
		"folder_open":  "▥",
		"file":         "▫",
		"file_code":    "◈",
		"file_text":    "▫",
		"file_config":  "●",
		"file_archive": "▦",
		"file_image":   "◉",
		"file_audio":   "♪",
		"file_video":   "▶",
		"success":      "✓",
		"error":        "✗",
		"warning":      "!",
		"info":         "i",
		"check":        "✓",
		"cross":        "✗",
		"arrow_up":     "↑",
		"arrow_down":   "↓",
		"arrow_left":   "←",
		"arrow_right":  "→",
		"loading":      "◐",
		"search":       "◎",
		"settings":     "●",
		"home":         "▣",
		"clinical":     "◆",
		"medical":      "◇",
		"pipeline":     "▥",
		"dictionary":   "▦",
		"analyze":      "◈",
		"document":     "▫",
		"cpu":          "◓",
		"memory":       "◒",
		"disk":         "◐",
		// Dictionary Builder specific icons
		"dict_main":     "▦",
		"dict_create":   "+",
		"dict_load":     "▤",
		"dict_config":   "●",
		"dict_tui":      "◈",
		"dict_vocab":    "▫",
		"dict_build":    "◆",
		"dict_template": "▫",
		"dict_memory":   "◐",
		"dict_process":  "◎",
		"dict_filter":   "◎",
		"dict_output":   "▶",
		"dict_relation": "◇",
		"dict_cased":    "◈",
	}
	// ASCII fallback icons (for basic terminals)
	asciiIcons = map[string]string{
		"folder":       "[D]",
		"folder_open":  "[+]",
		"file":         "[F]",
		"file_code":    "[C]",
		"file_text":    "[T]",
		"file_config":  "[*]",
		"file_archive": "[Z]",
		"file_image":   "[I]",
		"file_audio":   "[A]",
		"file_video":   "[V]",
		"success":      "[OK]",
		"error":        "[ERR]",
		"warning":      "[!]",
		"info":         "[i]",
		"check":        "[✓]",
		"cross":        "[X]",
		"arrow_up":     "^",
		"arrow_down":   "v",
		"arrow_left":   "<",
		"arrow_right":  ">",
		"loading":      "[~]",
		"search":       "[?]",
		"settings":     "[*]",
		"home":         "[H]",
		"clinical":     "[+]",
		"medical":      "[M]",
		"pipeline":     "[P]",
		"dictionary":   "[D]",
		"analyze":      "[A]",
		"document":     "[-]",
		"cpu":          "[CPU]",
		"memory":       "[MEM]",
		"disk":         "[DSK]",
		// Dictionary Builder specific icons
		"dict_main":     "[D]",
		"dict_create":   "[+]",
		"dict_load":     "[L]",
		"dict_config":   "[*]",
		"dict_tui":      "[T]",
		"dict_vocab":    "[V]",
		"dict_build":    "[B]",
		"dict_template": "[T]",
		"dict_memory":   "[M]",
		"dict_process":  "[P]",
		"dict_filter":   "[F]",
		"dict_output":   "[O]",
		"dict_relation": "[R]",
		"dict_cased":    "[C]",
	}

	// Current active icon set
	activeIcons map[string]string
)

// InitIcons initializes the icon system based on terminal capabilities
func InitIcons() {
	if initialized {
		return
	}
	initialized = true

	// Check env vars
	if os.Getenv("CTAKES_TUI_EMOJI") == "1" {
		supportsEmoji = true
		activeIcons = emojiIcons
		return
	}

	if os.Getenv("CTAKES_TUI_ASCII") == "1" {
		supportsEmoji = false
		activeIcons = asciiIcons
		return
	}

	// Auto-detect terminal
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	wtSession := os.Getenv("WT_SESSION")

	// Check emoji support
	if wtSession != "" || // Windows Terminal
		termProgram == "vscode" || // VS Code terminal
		termProgram == "iTerm.app" || // iTerm2
		termProgram == "Apple_Terminal" || // macOS Terminal
		strings.Contains(term, "256color") || // Most modern terminals
		strings.Contains(term, "alacritty") || // Alacritty
		strings.Contains(term, "kitty") { // Kitty
		supportsEmoji = true
		activeIcons = emojiIcons
	} else if runtime.GOOS == "windows" {
		// Windows console - limited emoji
		if term == "" || term == "dumb" {
			activeIcons = asciiIcons
		} else {
			activeIcons = unicodeIcons
		}
	} else {
		// Default to Unicode on Unix
		activeIcons = unicodeIcons
	}
}

// GetIcon returns an icon for the given key
func GetIcon(key string) string {
	if !initialized {
		InitIcons()
	}

	if icon, ok := activeIcons[key]; ok {
		return icon
	}

	// Fallback to empty
	return ""
}

// GetFileIcon returns an appropriate icon for a file or directory
func GetFileIcon(name string, isDir bool) string {
	if !initialized {
		InitIcons()
	}

	if isDir {
		return GetIcon("folder")
	}

	// Detect by extension
	ext := ""
	if idx := strings.LastIndex(name, "."); idx != -1 {
		ext = strings.ToLower(name[idx+1:])
	}

	// Map extensions
	switch ext {
	case "go", "js", "ts", "py", "java", "c", "cpp", "rs", "rb", "php":
		return GetIcon("file_code")
	case "txt", "md", "rst", "doc", "docx", "pdf":
		return GetIcon("file_text")
	case "json", "yaml", "yml", "toml", "ini", "conf", "config", "xml":
		return GetIcon("file_config")
	case "zip", "tar", "gz", "bz2", "xz", "rar", "7z":
		return GetIcon("file_archive")
	case "jpg", "jpeg", "png", "gif", "bmp", "svg", "ico":
		return GetIcon("file_image")
	case "mp3", "wav", "flac", "ogg", "m4a", "aac":
		return GetIcon("file_audio")
	case "mp4", "avi", "mkv", "mov", "wmv", "flv", "webm":
		return GetIcon("file_video")
	default:
		return GetIcon("file")
	}
}

// GetStatusIcon returns an icon for status indicators
func GetStatusIcon(status string) string {
	if !initialized {
		InitIcons()
	}

	switch strings.ToLower(status) {
	case "success", "ok", "done":
		return GetIcon("success")
	case "error", "fail", "failed":
		return GetIcon("error")
	case "warning", "warn":
		return GetIcon("warning")
	case "info":
		return GetIcon("info")
	case "loading", "pending":
		return GetIcon("loading")
	default:
		return ""
	}
}

// IsEmojiSupported returns whether the terminal supports emoji
func IsEmojiSupported() bool {
	if !initialized {
		InitIcons()
	}
	return supportsEmoji
}

// ForceIconMode allows manual override of icon mode
func ForceIconMode(mode string) {
	switch mode {
	case "emoji":
		activeIcons = emojiIcons
		supportsEmoji = true
	case "unicode":
		activeIcons = unicodeIcons
		supportsEmoji = false
	case "ascii":
		activeIcons = asciiIcons
		supportsEmoji = false
	}
	initialized = true
}
