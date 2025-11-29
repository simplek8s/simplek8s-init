// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package procfs

import (
	"testing"
)

func TestUnescape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// valid octal escapes
		{"/media/usb\\040stick", "/media/usb stick"},
		{"hello\\011world", "hello\tworld"},
		{"line1\\012line2", "line1\nline2"},

		// invalid sequences (should remain unchanged)
		{"bad\\09escape", "bad\\09escape"},         // not a valid 3-digit octal
		{"trailing\\0", "trailing\\0"},             // incomplete escape
		{"empty\\", "empty\\"},                     // just a backslash at end
		{"mixed\\040bad\\09ok", "mixed bad\\09ok"}, // partial parse

		// no escapes
		{"plainstring", "plainstring"},
	}

	for _, tt := range tests {
		got := unescape(tt.input)
		if got != tt.want {
			t.Errorf("unescape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseMounts(t *testing.T) {
	mockData := `/dev/sda1 / ext4 rw,relatime,errors=remount-ro 0 1
tmpfs /run tmpfs rw,nosuid,noexec,relatime,size=1638920k 0 0
invalid entry, must be skipped
/dev/sdb1 /media/usb\040stick vfat rw,nosuid,nodev,noexec 0 0`

	mounts, err := ParseMounts(mockData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check how many mounts the function returned.
	if len(mounts) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(mounts))
	}

	// Check first entry.
	if mounts[0].Source != "/dev/sda1" || mounts[0].Target != "/" || mounts[0].FSType != "ext4" {
		t.Errorf("unexpected first mount: %+v", mounts[0])
	}

	// Check unescape worked.
	if mounts[2].Target != "/media/usb stick" {
		t.Errorf("expected unescaped target '/media/usb stick', got %q", mounts[2].Target)
	}

	// Check options parsing.
	expectedOpt := "rw"
	if mounts[1].Options[0] != expectedOpt {
		t.Errorf("expected first option %q, got %q", expectedOpt, mounts[1].Options[0])
	}
}
