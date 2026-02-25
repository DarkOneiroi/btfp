package tui

import (
	"strings"
	"testing"
)

func TestItemTitleIcons(t *testing.T) {
	tests := []struct {
		name     string
		item     item
		contains string
	}{
		{
			name:     "Staged item",
			item:     item{title: "song.mp3", selected: true},
			contains: "󰄲",
		},
		{
			name:     "In playlist item",
			item:     item{title: "song.mp3", inPlaylist: true},
			contains: "󰄵",
		},
		{
			name:     "Directory item",
			item:     item{title: "Album", isDir: true},
			contains: "󰉋",
		},
		{
			name:     "Regular file item",
			item:     item{title: "song.mp3"},
			contains: "󰈣",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.Title()
			if !strings.Contains(got, tt.contains) {
				t.Errorf("item.Title() = %q, want it to contain %q", got, tt.contains)
			}
		})
	}
}
