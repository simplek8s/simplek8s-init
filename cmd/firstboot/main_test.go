package main

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	var got string
	var want string

	// Fallback
	os.Unsetenv("TESTING")
	got = getEnv("TESTING", "empty")
	want = "empty"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Value
	os.Setenv("TESTING", "something")
	got = getEnv("TESTING", "anotherthing")
	want = "something"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetOperation(t *testing.T) {
	tdd := []struct {
		args []string
		want int
	}{
		{
			args: []string{
				os.Args[0],
				"posibleDirectory",
				"posibleDirectory",
				"posibleDirectory",
			},
			want: OPERATION_SYSTEMD,
		},
		{
			args: []string{
				os.Args[0],
				"populate",
				"posibleDirectory",
			},
			want: OPERATION_POPULATE,
		},
		{
			args: []string{
				os.Args[0],
				"configurator",
				"posibleDirectory",
			},
			want: OPERATION_CONFIGURATOR,
		},
		{
			args: []string{
				os.Args[0],
			},
			want: OPERATION_NONE,
		},
	}

	for _, tc := range tdd {
		got := getOperation(tc.args)
		if got != tc.want {
			t.Errorf("got %d, want %d", got, tc.want)
		}
	}
}
