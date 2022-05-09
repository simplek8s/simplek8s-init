package yaml

import (
	"reflect"
	"testing"

	"github.com/openlyinc/pointy"
)

func Test_unmarshal(t *testing.T) {
	type args struct {
		yamlContent []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *SimpleK8s
		wantErr bool
	}{
		{
			name: "static ip network",
			args: args{
				yamlContent: []byte(`
storage:
  files:
    - path: /etc/systemd/network/50-en-static.conf
      content: |
        [Match]
        Name=en*

        [Network]
        Address=192.168.1.50/24
        Gateway=192.168.1.1
        DNS=8.8.8.8
`),
			},
			want: &SimpleK8s{
				Storage: simpleK8sStorage{
					Files: []simpleK8sFiles{
						{
							Path: "/etc/systemd/networkd/50-en-static.conf",
							Content: pointy.String(`[Match]
Name=en*

[Network]
Address=192.168.1.50/24
Gateway=192.168.1.1
DNS=8.8.8.8
`),
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := unmarshal(tt.args.yamlContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("unmarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}
