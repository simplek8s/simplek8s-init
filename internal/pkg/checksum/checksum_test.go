package checksum

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestSha256sum(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    string
		isEqual bool
		wantErr bool
	}{
		{
			name:    "ok text",
			args:    args{r: strings.NewReader("hello world")},
			want:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			isEqual: true,
			wantErr: false,
		},
		{
			name:    "ok binary",
			args:    args{r: bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})},
			want:    "1f825aa2f0020ef7cf91dfa30da4668d791c5d4824fc8e41354b89ec05795ab3",
			isEqual: true,
			wantErr: false,
		},
		{
			name:    "empty",
			args:    args{r: bytes.NewReader([]byte{})},
			want:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			isEqual: true,
			wantErr: false,
		},
		{
			name:    "ko",
			args:    args{r: strings.NewReader("bad result")},
			want:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			isEqual: false,
			wantErr: false,
		},
		{
			name:    "nil",
			args:    args{r: nil},
			want:    "",
			isEqual: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Sha256sum(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sha256sum() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (got == tt.want) != tt.isEqual {
				t.Errorf("Sha256sum() = %v, want %v", got, tt.want)
				return
			}
			if (got != tt.want) == tt.isEqual {
				t.Errorf("Sha256sum() = %v, do not want %v", got, tt.want)
			}
		})
	}
}
