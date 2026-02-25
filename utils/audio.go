package utils

import (
	"strings"
	"path/filepath"
)

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

func IsSupportedAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, supported := range SupportedExtensions {
		if ext == supported {
			return true
		}
	}
	return false
}
