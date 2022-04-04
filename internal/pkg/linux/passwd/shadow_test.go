package passwd

import (
	"fmt"
	"math"
	"reflect"
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
	shadow := NewShadow(Shadow{Name: "user"})

	//TODO: Validate required fields

	// Validate filled fields
	if len(shadow.Password) == 0 {
		t.Errorf("Shadow.FillDefaults() password = %v, password must be filled", shadow.Password)
	}
	if shadow.LastChanged == nil {
		t.Errorf("Shadow.FillDefaults() lastChanged = %v, lastChanged must be filled", shadow.LastChanged)
	}
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
		name: "ko name is required",
		shadow: Shadow{
			Password: "!!",
		},
		shadowAsString: "",
		wantErr:        true,
	},
	{
		name: "ko password is required",
		shadow: Shadow{
			Name: "user",
		},
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
	// tests := []testShadow{
	// 	{
	// 		name:           "ok",
	// 		shadow:         Shadow{},
	// 		shadowAsString: "user:$6$ejpbGffFgwYbUM6w$MW33ZDY3ObdVdN6Jxrj4TFZrO0wKKW65xE..2Z/nn8mMlIkIK6AThTW4JL6ZibKK2S5/0leA9UVJLNVzjaCLh0:18381:0:99999:7:::",
	// 		wantErr:        false,
	// 	},
	// }

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
