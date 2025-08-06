package utils

import "strings"

func GetFileIcon(name string, isDir bool) string {
	if isDir {
		return "📁"
	}

	ext := ""
	if idx := strings.LastIndex(name, "."); idx != -1 {
		ext = strings.ToLower(name[idx+1:])
	}

	switch ext {
	case "txt", "md", "doc", "docx", "pdf":
		return "📄"
	case "xml", "json", "yaml", "yml":
		return "📋"
	case "go", "java", "py", "js", "ts", "cpp", "c":
		return "💻"
	case "jpg", "jpeg", "png", "gif", "bmp":
		return "🖼️"
	case "mp3", "wav", "flac", "aac":
		return "🎵"
	case "mp4", "avi", "mkv", "mov":
		return "🎬"
	case "zip", "tar", "gz", "rar":
		return "📦"
	default:
		return "📄"
	}
}
