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

func TestNewGroup(t *testing.T) {
	t.Run("should initialize empty UserList when nil", func(t *testing.T) {
		g := NewGroup(Group{Name: "test", GID: 100})
		if g.UserList == nil {
			t.Fatal("expected UserList to be initialized to empty slice, got nil")
		}
		if len(g.UserList) != 0 {
			t.Fatalf("expected empty UserList, got %v", g.UserList)
		}
	})

	t.Run("should preserve existing UserList", func(t *testing.T) {
		users := []string{"alice", "bob"}
		g := NewGroup(Group{Name: "grp", GID: 200, UserList: users})
		if !reflect.DeepEqual(g.UserList, users) {
			t.Fatalf("expected %v, got %v", users, g.UserList)
		}
	})
}

func TestMarshal(t *testing.T) {
	tests := []struct {
		name      string
		group     Group
		want      string
		expectErr bool
	}{
		{
			name: "normal case",
			group: Group{
				Name:     "wheel",
				Password: "x",
				GID:      0,
				UserList: []string{"root", "admin"},
			},
			want: "wheel:x:0:root,admin",
		},
		{
			name: "empty password replaced with x",
			group: Group{
				Name:     "users",
				GID:      100,
				UserList: []string{},
			},
			want: "users:x:100:",
		},
		{
			name:      "missing name returns error",
			group:     Group{Password: "x", GID: 100},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.group.Marshal()
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

func TestUnmarshalGroup(t *testing.T) {
	tests := []struct {
		name      string
		entry     string
		want      Group
		expectErr bool
	}{
		{
			name:  "valid entry with users",
			entry: "wheel:x:0:root,admin",
			want: Group{
				Name:     "wheel",
				Password: "x",
				GID:      0,
				UserList: []string{"root", "admin"},
			},
		},
		{
			name:  "valid entry with empty user list",
			entry: "users:x:100:",
			want: Group{
				Name:     "users",
				Password: "x",
				GID:      100,
				UserList: []string{""},
			},
		},
		{
			name:      "invalid gid returns error",
			entry:     "broken:x:notint:root",
			expectErr: true,
		},
		{
			name:  "extra fields ignored",
			entry: "extra:x:200:foo,bar,unused",
			want: Group{
				Name:     "extra",
				Password: "x",
				GID:      200,
				UserList: []string{"foo", "bar", "unused"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Group
			err := UnmarshalGroup(tt.entry, &got)

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
