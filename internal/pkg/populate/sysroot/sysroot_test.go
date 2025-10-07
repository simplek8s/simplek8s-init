package sysroot

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlsalvador/simplek8s/internal/pkg/sysroot"
)

func Test_populateSysrootSshd(t *testing.T) {
	output := t.TempDir()

	var want = map[bool][]byte{
		true:  []byte("PasswordAuthentication yes\nPermitRootLogin yes\nAllowUsers root\n"),
		false: []byte("PasswordAuthentication no\nPermitRootLogin prohibit-password\nAllowUsers root\n"),
	}

	for _, allowRootPassword := range []bool{true, false} {
		tname := fmt.Sprintf("allowRootPassword %t", allowRootPassword)
		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			sr := sysroot.Sysroot{}

			if err := populateSysrootSshd(output, &sr, allowRootPassword); err != nil {
				t.Fatal(err)
			}

			dst := filepath.Join(output, "/usr/share/factory")
			if err := os.MkdirAll(dst, 0755); err != nil {
				t.Fatal(err)
			}

			if err := sr.Write(dst); err != nil {
				t.Fatal(err)
			}

			dst = filepath.Join(output, "/usr/share/factory/etc/ssh/sshd_config")
			if content, err := os.ReadFile(dst); err != nil {
				t.Fatal(err)
			} else if !bytes.Equal(content, want[allowRootPassword]) {
				t.Fatalf("got: %v, want: %v", content, want[allowRootPassword])
			}
		})
	}
}
