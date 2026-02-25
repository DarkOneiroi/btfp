package utils

import (
	"path/filepath"
	"strings"
)

// SupportedExtensions is a list of audio file extensions supported by btfp
var SupportedExtensions = []string{
	".mp3",
	".wav",
	".flac",
	".ogg",
	".m4a",
	".aac",
	".wma",
	".aiff",
	".opus",
}

// IsSupportedAudioFile returns true if the file extension is in the supported list
func IsSupportedAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range SupportedExtensions {
		if ext == supported {
			return true
		}
	}
	return false
}
