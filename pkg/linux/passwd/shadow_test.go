// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package passwd

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/openlyinc/pointy"
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
	fields := []string{
		"LastChanged", "Minimum", "Maximum", "Warn", "Inactive", "Expire", "Reserved",
	}

	validValue := "123"
	invalidValue := "abc"

	for i, field := range fields {
		t.Run(fmt.Sprintf("%s_empty", field), func(t *testing.T) {
			tokens := make([]string, 9)
			tokens[0] = "user"
			tokens[1] = "pass"
			// el campo i+2 será vacío (ya lo está por default)
			entry := strings.Join(tokens, ":")
			var s Shadow
			if err := UnmarshalShadow(entry, &s); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// el campo correspondiente debe ser nil
			switch field {
			case "LastChanged":
				if s.LastChanged != nil {
					t.Fatalf("expected nil, got %v", *s.LastChanged)
				}
			case "Minimum":
				if s.Minimum != nil {
					t.Fatalf("expected nil, got %v", *s.Minimum)
				}
			case "Maximum":
				if s.Maximum != nil {
					t.Fatalf("expected nil, got %v", *s.Maximum)
				}
			case "Warn":
				if s.Warn != nil {
					t.Fatalf("expected nil, got %v", *s.Warn)
				}
			case "Inactive":
				if s.Inactive != nil {
					t.Fatalf("expected nil, got %v", *s.Inactive)
				}
			case "Expire":
				if s.Expire != nil {
					t.Fatalf("expected nil, got %v", *s.Expire)
				}
			case "Reserved":
				if s.Reserved != nil {
					t.Fatalf("expected nil, got %v", *s.Reserved)
				}
			}
		})

		t.Run(fmt.Sprintf("%s_invalid", field), func(t *testing.T) {
			tokens := make([]string, 9)
			tokens[0] = "user"
			tokens[1] = "pass"
			tokens[i+2] = invalidValue
			entry := strings.Join(tokens, ":")
			var s Shadow
			if err := UnmarshalShadow(entry, &s); err == nil {
				t.Fatal("expected error, got nil")
			}
		})

		t.Run(fmt.Sprintf("%s_valid", field), func(t *testing.T) {
			tokens := make([]string, 9)
			tokens[0] = "user"
			tokens[1] = "pass"
			tokens[i+2] = validValue
			entry := strings.Join(tokens, ":")
			var s Shadow
			if err := UnmarshalShadow(entry, &s); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			expected := 123
			switch field {
			case "LastChanged":
				if s.LastChanged == nil || *s.LastChanged != expected {
					t.Fatalf("expected %d, got %v", expected, s.LastChanged)
				}
			case "Minimum":
				if s.Minimum == nil || *s.Minimum != expected {
					t.Fatalf("expected %d, got %v", expected, s.Minimum)
				}
			case "Maximum":
				if s.Maximum == nil || *s.Maximum != expected {
					t.Fatalf("expected %d, got %v", expected, s.Maximum)
				}
			case "Warn":
				if s.Warn == nil || *s.Warn != expected {
					t.Fatalf("expected %d, got %v", expected, s.Warn)
				}
			case "Inactive":
				if s.Inactive == nil || *s.Inactive != expected {
					t.Fatalf("expected %d, got %v", expected, s.Inactive)
				}
			case "Expire":
				if s.Expire == nil || *s.Expire != expected {
					t.Fatalf("expected %d, got %v", expected, s.Expire)
				}
			case "Reserved":
				if s.Reserved == nil || *s.Reserved != expected {
					t.Fatalf("expected %d, got %v", expected, s.Reserved)
				}
			}
		})
	}
}
