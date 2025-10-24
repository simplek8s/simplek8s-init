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
	"reflect"
	"testing"
)

func TestNewUser(t *testing.T) {
	t.Run("should set default home and shell", func(t *testing.T) {
		u := NewUser(User{Name: "sysuser"})
		if u.Home != "/" {
			t.Errorf("expected default home '/', got %q", u.Home)
		}
		if u.Shell != "/usr/sbin/nologin" {
			t.Errorf("expected default shell '/usr/sbin/nologin', got %q", u.Shell)
		}
	})

	t.Run("should preserve custom home and shell", func(t *testing.T) {
		u := NewUser(User{Name: "jdoe", Home: "/home/jdoe", Shell: "/bin/bash"})
		if u.Home != "/home/jdoe" {
			t.Errorf("expected home '/home/jdoe', got %q", u.Home)
		}
		if u.Shell != "/bin/bash" {
			t.Errorf("expected shell '/bin/bash', got %q", u.Shell)
		}
	})
}

func TestMarshalUser(t *testing.T) {
	tests := []struct {
		name      string
		user      User
		want      string
		expectErr bool
	}{
		{
			name: "normal user",
			user: User{
				Name:     "jdoe",
				Password: "x",
				UID:      1000,
				GID:      1000,
				Gecos:    []string{"John Doe"},
				Home:     "/home/jdoe",
				Shell:    "/bin/bash",
			},
			want: "jdoe:x:1000:1000:John Doe:/home/jdoe:/bin/bash",
		},
		{
			name: "defaults applied when password/home/shell missing",
			user: User{
				Name:  "sysuser",
				UID:   1,
				GID:   1,
				Gecos: []string{},
			},
			want: "sysuser:x:1:1::/:/usr/bin/nologin",
		},
		{
			name:      "missing name returns error",
			user:      User{Password: "x", UID: 100, GID: 100},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.user.Marshal()
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestUnmarshalUser(t *testing.T) {
	tests := []struct {
		name      string
		entry     string
		want      User
		expectErr bool
	}{
		{
			name:  "valid entry with all fields",
			entry: "jdoe:x:1000:1000:John Doe:/home/jdoe:/bin/bash",
			want: User{
				Name:     "jdoe",
				Password: "x",
				UID:      1000,
				GID:      1000,
				Gecos:    []string{"John Doe"},
				Home:     "/home/jdoe",
				Shell:    "/bin/bash",
			},
		},
		{
			name:  "entry with empty GECOS and shell",
			entry: "sysuser:x:1:1::/:/usr/sbin/nologin",
			want: User{
				Name:     "sysuser",
				Password: "x",
				UID:      1,
				GID:      1,
				Gecos:    []string{""},
				Home:     "/",
				Shell:    "/usr/sbin/nologin",
			},
		},
		{
			name:      "invalid UID returns error",
			entry:     "broken:x:notint:100::/:/bin/sh",
			expectErr: true,
		},
		{
			name:      "invalid GID returns error",
			entry:     "broken:x:100:notint::/:/bin/sh",
			expectErr: true,
		},
		{
			name:  "extra fields ignored gracefully",
			entry: "extra:x:2000:2000:Info:/tmp:/bin/zsh:ignored",
			want: User{
				Name:     "extra",
				Password: "x",
				UID:      2000,
				GID:      2000,
				Gecos:    []string{"Info"},
				Home:     "/tmp",
				Shell:    "/bin/zsh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got User
			err := UnmarshalUser(tt.entry, &got)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}
