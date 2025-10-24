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

package common

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	var got string
	var want string

	// Fallback
	os.Unsetenv("TESTING")
	got = GetEnv("TESTING", "empty")
	want = "empty"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Value
	os.Setenv("TESTING", "something")
	got = GetEnv("TESTING", "anotherthing")
	want = "something"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGet(t *testing.T) {
	t.Run("bool pointer is nil", func(t *testing.T) {
		var ptr *bool
		result := Get(ptr, true)
		if result != true {
			t.Errorf("expected true, got %v", result)
		}
	})

	t.Run("bool pointer is not nil", func(t *testing.T) {
		val := false
		result := Get(&val, true)
		if result != false {
			t.Errorf("expected false, got %v", result)
		}
	})

	t.Run("string pointer is nil", func(t *testing.T) {
		var name *string
		result := Get(name, "default")
		if result != "default" {
			t.Errorf("expected 'default', got %q", result)
		}
	})

	t.Run("string pointer is not nil", func(t *testing.T) {
		value := "hello"
		result := Get(&value, "default")
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})

	t.Run("int pointer is nil", func(t *testing.T) {
		var number *int
		result := Get(number, 42)
		if result != 42 {
			t.Errorf("expected 42, got %v", result)
		}
	})

	t.Run("int pointer is not nil", func(t *testing.T) {
		val := 99
		result := Get(&val, 0)
		if result != 99 {
			t.Errorf("expected 99, got %v", result)
		}
	})
}
