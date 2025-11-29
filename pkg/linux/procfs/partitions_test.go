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

package procfs_test

import (
	"reflect"
	"testing"

	"simplek8s/pkg/linux/procfs"
)

func TestParsePartitions_Success(t *testing.T) {
	content := `major minor  #blocks  name

 259        0 1953514584 nvme0n1
 259        4 1953513472 nvme0n1p1
   8        0  488386584 sda
   8        1  488386000 sda1`

	expected := []procfs.Partitions{
		{Major: 259, Minor: 0, Blocks: 1953514584, Name: "nvme0n1"},
		{Major: 259, Minor: 4, Blocks: 1953513472, Name: "nvme0n1p1"},
		{Major: 8, Minor: 0, Blocks: 488386584, Name: "sda"},
		{Major: 8, Minor: 1, Blocks: 488386000, Name: "sda1"},
	}

	result, err := procfs.ParsePartitions(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestParsePartitions_InvalidHeader(t *testing.T) {
	content := `invalid header line

 259        0 1953514584 nvme0n1`

	_, err := procfs.ParsePartitions(content)
	if err == nil {
		t.Fatal("expected error for invalid header, got nil")
	}

	expectedMsg := `unexpected header, got "invalid header line"`
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestParsePartitions_InvalidNewLineSeparator(t *testing.T) {
	content := `major minor  #blocks  name
not empty
 259        0 1953514584 nvme0n1`

	_, err := procfs.ParsePartitions(content)
	if err == nil {
		t.Fatal("expected error for invalid newline separator, got nil")
	}

	expectedMsg := `unexpected new line header separator, got "not empty"`
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestParsePartitions_InvalidFieldCount(t *testing.T) {
	content := `major minor  #blocks  name

 259        0 1953514584`

	_, err := procfs.ParsePartitions(content)
	if err == nil {
		t.Fatal("expected error for invalid field count, got nil")
	}

	if !contains(err.Error(), "cannot parse line") {
		t.Errorf("expected 'cannot parse line' error, got %q", err.Error())
	}
}

func TestParsePartitions_InvalidMajor(t *testing.T) {
	content := `major minor  #blocks  name

 abc        0 1953514584 nvme0n1`

	if _, err := procfs.ParsePartitions(content); err == nil {
		t.Fatal("expected error for invalid major number, got nil")
	}
}

func TestParsePartitions_InvalidMinor(t *testing.T) {
	content := `major minor  #blocks  name

 259      abc 1953514584 nvme0n1`

	if _, err := procfs.ParsePartitions(content); err == nil {
		t.Fatal("expected error for invalid minor number, got nil")
	}
}

func TestParsePartitions_InvalidBlocks(t *testing.T) {
	content := `major minor  #blocks  name

 259        0 abc nvme0n1`

	if _, err := procfs.ParsePartitions(content); err == nil {
		t.Fatal("expected error for invalid blocks number, got nil")
	}
}

func TestParsePartitions_EmptyFile(t *testing.T) {
	content := `major minor  #blocks  name
`
	result, err := procfs.ParsePartitions(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %+v", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
