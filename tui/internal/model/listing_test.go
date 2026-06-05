package model

import (
	"testing"
)

func TestParentPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", ""},
		{"/", ""},
		{"/group1", ""},
		{"/group1/entry1", "/group1"},
		{"/g1/g2/entry1", "/g1/g2"},
	}
	for _, tt := range tests {
		result := ParentPath(tt.path)
		if result != tt.expected {
			t.Errorf("ParentPath(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestEntryPath(t *testing.T) {
	tests := []struct {
		groupPath string
		entryID   string
		expected  string
	}{
		{"", "entry1", "/entry1"},
		{"/", "entry1", "/entry1"},
		{"group1", "entry1", "/group1/entry1"},
		{"/group1", "entry1", "/group1/entry1"},
		{"/group1/sub", "entry1", "/group1/sub/entry1"},
	}

	for _, tt := range tests {
		result := EntryPath(tt.groupPath, tt.entryID)
		if result != tt.expected {
			t.Errorf("EntryPath(%q, %q) = %q, want %q", tt.groupPath, tt.entryID, result, tt.expected)
		}
	}
}
