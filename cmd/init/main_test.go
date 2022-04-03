package main

import (
	"os"
	"os/exec"
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

func TestMain(t *testing.T) {
	t.Run("no arguments", func(t *testing.T) {
		if os.Getenv("BE_MAIN") == "1" {
			main()
			return
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestMain")
		cmd.Env = append(os.Environ(), "BE_MAIN=1")
		err := cmd.Run()
		if e, ok := err.(*exec.ExitError); ok && !e.Success() {
			return
		}

		t.Fatalf("process ran with err %v, want exit status 1", err)
	})

	t.Run("debug true", func(t *testing.T) {
		t.Setenv("DEBUG", "true")

		main()
	})
	t.Run("debug false", func(t *testing.T) {
		t.Setenv("DEBUG", "false")

		main()
	})
}
