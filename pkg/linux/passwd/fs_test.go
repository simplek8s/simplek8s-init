// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package passwd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"simplek8s/pkg/linux/passwd"

	"go.openly.dev/pointy"
)

func TestReadShadowFile_OK(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "shadow")
	os.WriteFile(tmp, []byte("root:!!:0::::::\n\nuser:!!:0::::::\n"), 0600)

	shadows, err := passwd.ReadShadowFile(tmp)
	if err != nil {
		t.Fatal(err)
	}

	if len(shadows) != 2 {
		t.Fatalf("expected 2 shadows, got %d", len(shadows))
	}
	if shadows[0].Name != "root" || shadows[1].Name != "user" {
		t.Fatalf("unexpected shadow contents: %+v", shadows)
	}
}

func TestReadShadowFile_Invalid(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "shadow")
	os.WriteFile(tmp, []byte("root:!!:0::::::\n\nuser\n"), 0600)

	_, err := passwd.ReadShadowFile(tmp)
	if err == nil || !strings.Contains(err.Error(), "invalid number of fields in entry") {
		t.Fatalf("expected error, got: %v", err)
	}
}

func TestWriteShadows_OldDoesNotExists(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "shadow")
	data := []passwd.Shadow{
		passwd.NewShadow(passwd.Shadow{
			Name:        "root",
			LastChanged: pointy.Int(0),
		}),
	}

	if err := passwd.WriteShadows(tmp, data); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatal(err)
	}

	expected := "root:!!:0::::::\n"
	if string(content) != expected {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestWriteShadows_Atomicity(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "shadow")
	data := []passwd.Shadow{
		passwd.NewShadow(passwd.Shadow{
			Name:        "root",
			LastChanged: pointy.Int(0),
		}),
	}
	os.WriteFile(tmp, []byte("old\n"), 0644)

	if err := passwd.WriteShadows(tmp, data); err != nil {
		t.Fatal(err)
	}

	want := "root:!!:0::::::\n"
	content, _ := os.ReadFile(tmp)
	if string(content) != want {
		t.Fatalf("expected %q, got %q", want, string(content))
	}
}

func TestWriteShadows_CantCreateTmp(t *testing.T) {
	tmpDir := t.TempDir()

	// Set directory as read-only.
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755)

	dst := filepath.Join(tmpDir, "shadow")
	if err := passwd.WriteShadows(dst, []passwd.Shadow{}); err == nil || !strings.Contains(err.Error(), "cannot create temp file") {
		t.Fatalf("expected error, got %v", err)
	}
}
