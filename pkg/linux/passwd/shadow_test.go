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

package passwd

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.openly.dev/pointy"
)

func TestShadow_GetNumberOfDaysFrom1970(t *testing.T) {
	var seconds, minutes, hours float64
	seconds = float64(time.Now().Unix())
	minutes = seconds / 60
	hours = minutes / 60
	days := math.Ceil(hours / 24)

	got := getNumberOfDaysFrom1970()

	if int(days) != got {
		t.Errorf("want: %f, got %d", days, got)
	}
}

func TestShadow_NewShadow(t *testing.T) {
	t.Run("panic if name empty", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for empty name, but did not panic")
			}
		}()
		NewShadow(Shadow{})
	})

	t.Run("password empty becomes !!", func(t *testing.T) {
		s := NewShadow(Shadow{Name: "user"})
		if s.Password != "!!" {
			t.Fatalf("expected password to be '!!', got '%s'", s.Password)
		}
	})

	t.Run("lastChanged nil gets initialized", func(t *testing.T) {
		s := NewShadow(Shadow{Name: "user"})
		if s.LastChanged == nil {
			t.Fatal("expected LastChanged to be initialized, got nil")
		}
		daysFrom1970 := getNumberOfDaysFrom1970()
		if *s.LastChanged != daysFrom1970 {
			t.Fatalf("expected LastChanged to be %d, got %d", daysFrom1970, *s.LastChanged)
		}
	})

	t.Run("keep existing password and lastChanged", func(t *testing.T) {
		lc := 12345
		s := Shadow{
			Name:        "user",
			Password:    "mypassword",
			LastChanged: &lc,
		}
		result := NewShadow(s)
		if result.Password != "mypassword" {
			t.Fatalf("expected password to remain 'mypassword', got '%s'", result.Password)
		}
		if *result.LastChanged != 12345 {
			t.Fatalf("expected LastChanged to remain 12345, got %d", *result.LastChanged)
		}
	})
}

var tests = []struct {
	name           string
	shadow         Shadow
	shadowAsString string
	wantErr        bool
}{
	{
		name: "ok full",
		shadow: Shadow{
			Name:        "user",
			Password:    "$6$ejpbGffFgwYbUM6w$MW33ZDY3ObdVdN6Jxrj4TFZrO0wKKW65xE..2Z/nn8mMlIkIK6AThTW4JL6ZibKK2S5/0leA9UVJLNVzjaCLh0",
			LastChanged: pointy.Int(18381),
			Minimum:     pointy.Int(0),
			Maximum:     pointy.Int(99999),
			Warn:        pointy.Int(7),
			Inactive:    pointy.Int(30),
			Expire:      pointy.Int(0),
			Reserved:    pointy.Int(0),
		},
		shadowAsString: "user:$6$ejpbGffFgwYbUM6w$MW33ZDY3ObdVdN6Jxrj4TFZrO0wKKW65xE..2Z/nn8mMlIkIK6AThTW4JL6ZibKK2S5/0leA9UVJLNVzjaCLh0:18381:0:99999:7:30:0:0",
		wantErr:        false,
	},
	{
		name: "ok NewShadow",
		shadow: NewShadow(Shadow{
			Name: "user",
		}),
		shadowAsString: fmt.Sprintf("user:!!:%d::::::", getNumberOfDaysFrom1970()),
		wantErr:        false,
	},
	{
		name:           "ko name is required",
		shadow:         Shadow{},
		shadowAsString: "",
		wantErr:        true,
	},
}

func TestMarshalShadow(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.shadow.Marshal()
			if tt.wantErr && err == nil {
				t.Errorf("got %q, err %q, expected error", got, err)
			} else if !tt.wantErr && err != nil {
				t.Errorf("got %q, err %q", got, err)
			} else if tt.shadowAsString != got {
				t.Errorf("want %q, got %q", tt.shadowAsString, got)
			}
		})
	}
}

func TestUnmarshalShadow(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Shadow{}
			err := UnmarshalShadow(tt.shadowAsString, &got)
			if tt.wantErr && err == nil {
				t.Errorf("got %#v, err %#v, expected error", got, err)
			} else if !tt.wantErr && err != nil {
				t.Errorf("got %#v, err %#v", got, err)
			} else if !reflect.DeepEqual(got, tt.shadow) {
				t.Errorf("got %#v, want %#v", got, tt.shadow)
			}
		})
	}
}

func TestUnmarshalShadow_AtoiCoverage(t *testing.T) {
	type shadowField struct {
		name string
		get  func(*Shadow) *int
	}

	fields := []shadowField{
		{"LastChanged", func(s *Shadow) *int { return s.LastChanged }},
		{"Minimum", func(s *Shadow) *int { return s.Minimum }},
		{"Maximum", func(s *Shadow) *int { return s.Maximum }},
		{"Warn", func(s *Shadow) *int { return s.Warn }},
		{"Inactive", func(s *Shadow) *int { return s.Inactive }},
		{"Expire", func(s *Shadow) *int { return s.Expire }},
		{"Reserved", func(s *Shadow) *int { return s.Reserved }},
	}

	assertNil := func(t *testing.T, v *int) {
		t.Helper()
		if v != nil {
			t.Fatalf("expected nil, got %v", *v)
		}
	}

	assertEqual := func(t *testing.T, v *int, expected int) {
		t.Helper()
		if v == nil || *v != expected {
			t.Fatalf("expected %d, got %v", expected, v)
		}
	}

	runEmpty := func(t *testing.T, f shadowField) {
		t.Helper()
		tokens := make([]string, 9)
		tokens[0], tokens[1] = "user", "pass"
		entry := strings.Join(tokens, ":")
		var s Shadow
		if err := UnmarshalShadow(entry, &s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertNil(t, f.get(&s))
	}

	runInvalid := func(t *testing.T, _ shadowField, idx int) {
		t.Helper()
		tokens := make([]string, 9)
		tokens[0], tokens[1] = "user", "pass"
		tokens[idx+2] = "abc"
		entry := strings.Join(tokens, ":")
		var s Shadow
		if err := UnmarshalShadow(entry, &s); err == nil {
			t.Fatal("expected error, got nil")
		}
	}

	runValid := func(t *testing.T, f shadowField, idx int) {
		t.Helper()
		tokens := make([]string, 9)
		tokens[0], tokens[1] = "user", "pass"
		tokens[idx+2] = "123"
		entry := strings.Join(tokens, ":")
		var s Shadow
		if err := UnmarshalShadow(entry, &s); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEqual(t, f.get(&s), 123)
	}

	for i, f := range fields {
		t.Run(f.name+"_empty", func(t *testing.T) { runEmpty(t, f) })
		t.Run(f.name+"_invalid", func(t *testing.T) { runInvalid(t, f, i) })
		t.Run(f.name+"_valid", func(t *testing.T) { runValid(t, f, i) })
	}
}
