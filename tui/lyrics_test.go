package tui

import (
	"btfp/player"
	"reflect"
	"testing"
	"time"
)

func TestParseLyrics(t *testing.T) {
	m := &Model{}

	tests := []struct {
		name         string
		content      string
		expected     []lrcLine
		expectSynced bool
	}{
		{
			name: "Synced lyrics",
			content: `[00:01.00] First line
[00:02.50] Second line`,
			expected: []lrcLine{
				{time: 1 * time.Second, text: "First line"},
				{time: 2500 * time.Millisecond, text: "Second line"},
			},
			expectSynced: true,
		},
		{
			name: "Synced lyrics with tags",
			content: `[ti:Test]
[00:01.00] First line
[00:02.00] Second line`,
			expected: []lrcLine{
				{time: 1 * time.Second, text: "First line"},
				{time: 2 * time.Second, text: "Second line"},
			},
			expectSynced: true,
		},
		{
			name: "Plain lyrics",
			content: `First line
Second line`,
			expected: []lrcLine{
				{time: 0 * time.Second, text: "First line"},
				{time: 5 * time.Second, text: "Second line"},
			},
			expectSynced: false,
		},
		{
			name: "Mixed (should include non-timestamped lines with previous timestamp)",
			content: `[00:01.00] First line
Random line
[00:02.00] Second line`,
			expected: []lrcLine{
				{time: 1 * time.Second, text: "First line"},
				{time: 1 * time.Second, text: "Random line"},
				{time: 2 * time.Second, text: "Second line"},
			},
			expectSynced: true,
		},
		{
			name:    "Lyrics with different line endings",
			content: "[00:01.00] Line 1\r\n[00:02.00] Line 2\r[00:03.00] Line 3",
			expected: []lrcLine{
				{time: 1 * time.Second, text: "Line 1"},
				{time: 2 * time.Second, text: "Line 2"},
				{time: 3 * time.Second, text: "Line 3"},
			},
			expectSynced: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, synced := m.parseLyrics(tt.content)
			if synced != tt.expectSynced {
				t.Errorf("parseLyrics() synced = %v, want %v", synced, tt.expectSynced)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseLyrics() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCleanString(t *testing.T) {
	// Manual check for complex cases
	check := func(in, exp string) {
		if got := cleanString(in); got != exp {
			t.Errorf("cleanString(%q) = [%s], want [%s]", in, got, exp)
		}
	}

	check("01 - Master of Puppets", "master of puppets")
	check("10 Wasting My Hate", "wasting my hate")
	check("Enter Sandman (Official Video)", "enter sandman")
	check("Nothing Else Matters [Remastered]", "nothing else matters")
	check("Some Song (Live at Wembley)", "some song")
	check("  Excessive   Spaces  ", "excessive spaces")
	check("01 - Song", "song")
	check("Song (Official)", "song")
	check("Song [Lyrics]", "song")
	check("Song 4K HD", "song")
	check("  Space  ", "space")
	check("King Nothing", "king nothing")
}

func TestPlaylistCounts(t *testing.T) {
	m := &Model{
		playlist: []player.Track{
			{Path: "/home/user/Music/A/1.mp3"},
			{Path: "/home/user/Music/A/2.mp3"},
			{Path: "/home/user/Music/B/1.mp3"},
		},
	}

	m.updatePlaylistCounts()

	if m.playlistCounts["/home/user/Music/A/1.mp3"] != 1 {
		t.Errorf("Track count wrong for file, got %d", m.playlistCounts["/home/user/Music/A/1.mp3"])
	}

	if m.playlistCounts["/home/user/Music/A"] != 2 {
		t.Errorf("Dir count wrong for A, got %d", m.playlistCounts["/home/user/Music/A"])
	}

	if m.playlistCounts["/home/user/Music"] != 3 {
		t.Errorf("Dir count wrong for Music, got %d", m.playlistCounts["/home/user/Music"])
	}
}
