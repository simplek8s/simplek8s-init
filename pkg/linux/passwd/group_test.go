// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package passwd

import (
	"reflect"
	"testing"
)

type testGroup struct {
	name          string
	group         Group
	groupAsString string
	wantErr       bool
}

func TestMarshalGroup(t *testing.T) {
	tests := []testGroup{
		{
			name: "ok",
			group: Group{
				Name:     "users",
				Password: "x",
				Gid:      985,
				UserList: []string{
					"john",
					"user",
				},
			},
			groupAsString: "users:x:985:john,user",
			wantErr:       false,
		},
		{
			name:          "empty name",
			group:         Group{},
			groupAsString: "",
			wantErr:       true,
		},
		{
			name: "just name",
			group: Group{
				Name: "users",
			},
			groupAsString: "users:x:0:",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.group.Marshal()
			if tt.wantErr && err == nil {
				t.Errorf("got %q, err %q, expected error", got, err)
			} else if !tt.wantErr && err != nil {
				t.Errorf("got %q, err %q", got, err)
			} else if tt.groupAsString != got {
				t.Errorf("want %q, got %q", tt.groupAsString, got)
			}
		})
	}
}

func TestUnmarshalGroup(t *testing.T) {
	tests := []testGroup{
		{
			name: "ok",
			group: Group{
				Name:     "users",
				Password: "x",
				Gid:      985,
				UserList: []string{
					"john",
					"user",
				},
			},
			groupAsString: "users:x:985:john,user",
			wantErr:       false,
		},
		//TODO: KO tests
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Group{}
			err := UnmarshalGroup(tt.groupAsString, &got)
			if tt.wantErr && err == nil {
				t.Errorf("got %v, err %q, expected error", got, err)
			} else if !tt.wantErr && err != nil {
				t.Errorf("got %v, err %q", got, err)
			} else if !reflect.DeepEqual(got, tt.group) {
				t.Errorf("got %v, want %v", got, tt.group)
			}
		})
	}
}
