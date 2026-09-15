// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package bootstrap_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
	"github.com/simplek8s/simplek8s-init/pkg/linux/passwd"
	"github.com/simplek8s/simplek8s-init/pkg/simplek8s/bootstrap"
	sr "github.com/simplek8s/simplek8s-init/pkg/simplek8s/sysroot"

	"go.openly.dev/pointy"
)

func TestFeedSysrootByBootstrapConfig(t *testing.T) {
	config := bootstrap.Config{
		Version: bootstrap.ConfigVersion1,
		Groups: []bootstrap.Group{
			{
				GID:    pointy.Int(985),
				Name:   "users",
				System: pointy.Bool(true),
			},
			{
				Name:   "video",
				System: pointy.Bool(true),
			},
			{
				Name:   "dev_user",
				System: pointy.Bool(false),
			},
		},
		Users: []bootstrap.User{
			{
				UID:          pointy.Int(0),
				GID:          pointy.Int(0),
				Name:         "root",
				PasswordHash: pointy.String("$6$vV10BRFDn5d3U4.w$9lEgwXZSjXvfqTa161UucWRbTlj53LlokWQ0GYpac3Ralola5UFHsKQ7Xbl7SuWuCuSv19usBGLOXxTkLLgv91"),
				SSHAuthorizedKeys: []string{
					"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICOJHHXFdcBLMAviMAHgQvCuzpnLmzXxatIL6IUe7b6W salvador.joseluis@gmail.com",
					"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC/F0iA8E/95yYMLAyRTGgW9VFQgljQPcX3rL+nE/mjw9jLh/1OtHM/Glu+kkXwYsOv06tCJOIGsUX0itjaAiGU1BaZa2NyUlHBgHzh7MmF1cNKdRK8+NTO7evLzQOMYPOP8rVuoyYhaBjXc5amP5gMndQnZkabP6wbjvbi+rldFE0hmXLaj0rI8eJI/hjk/ra+cH+kaqx93eZqTL4WRdHw821DESSbiU9m8hUJif4AyhKw41iKosvUTVDmoJ9O5Ed6Rid+hN1DUei1QF8sXjrPxBChKQSMEvnU5XBmFUpxi9VL85kVZVGKKdm7D9VHsI2fIMexMV7BbdP62636T3wKgl9fgn1QGYsVZ3eEbDVIK0WY+wJB9b7o3MS+Os9tT4mXO27BN4cCEuV6Mgbt0Qt1YFG/jVaVSTuecOoHkcCrue8tseqEKE4qwJvL73NJ8pxjBml2L1m/unQaBlbCJpkpgLFv1GS4C8NEnZwJfC9BoXvo/TfDMbTr1zdxT+hQqubpizqq0KeWFOPojGMRhdri0fXUuPkpaqQr95KAJ5q6Awp7AOC/v4tPCePPW6clzHspITVmEGpGUTErD2ASrChGNkr/7s+nICqpK5qQRdvZHRUvfa/WUPsrpFiR6cnRPM/dpe771chVSf6e4U5KyOHduGBhuXPFaL17O22cjsG7aQ== salvador.joseluis@gmail.com",
				},
				Groups: []string{
					"users",
				},
				System: pointy.Bool(false),
				Gecos:  pointy.String("Super User"),
			},
			{
				Name: "rpi",
				SSHAuthorizedKeys: []string{
					"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDnSPJEpeYRLtS0ebiKuFFu2owz4Jwz7vQZ0FcAUARbb opencode@opencode-test",
				},
				Groups: []string{
					"users",
					"dev_user",
				},
			},
		},
		Storage: &bootstrap.Storage{
			Mounts: []bootstrap.Mount{
				{
					What:    "/dev/disk/by-label/var",
					Where:   "/var",
					Type:    pointy.String("ext4"),
					Options: pointy.String("rw,relatime,discard"),
					After:   nil,
				},
				{
					What:    "/dev/disk/by-label/etc",
					Where:   "/etc",
					Type:    nil,
					Options: nil,
					After:   nil,
				},
			},
		},
	}

	want := &sr.Sysroot{
		Shadows: []passwd.Shadow{
			passwd.NewShadow(passwd.Shadow{
				Name:     "root",
				Password: "$6$vV10BRFDn5d3U4.w$9lEgwXZSjXvfqTa161UucWRbTlj53LlokWQ0GYpac3Ralola5UFHsKQ7Xbl7SuWuCuSv19usBGLOXxTkLLgv91",
			}),
			// Keys without password hash must unlock the account
			// ("*") instead of leaving it "!unprovisioned", or
			// sshd would refuse even public key authentication.
			passwd.NewShadow(passwd.Shadow{
				Name:     "rpi",
				Password: "*",
			}),
		},
		Groups: []passwd.Group{
			{
				Name:     "root",
				Password: "",
				GID:      0,
				UserList: []string{
					"root",
				},
			},
			{
				Name:     "users",
				Password: "",
				GID:      985,
				UserList: []string{
					"root",
					"rpi",
				},
			},
			{
				Name:     "video",
				Password: "",
				GID:      999,
				UserList: []string{},
			},
			{
				Name:     "rpi",
				Password: "",
				GID:      1000,
				UserList: []string{
					"rpi",
				},
			},
			{
				Name:     "dev_user",
				Password: "",
				GID:      1001,
				UserList: []string{
					"rpi",
				},
			},
		},
		Users: []passwd.User{
			{
				Name:     "root",
				Password: "x",
				UID:      0,
				GID:      0,
				Gecos:    "Super User",
				Home:     "/root",
				Shell:    "/usr/bin/sh",
			},
			{
				Name:     "rpi",
				Password: "x",
				UID:      1000,
				GID:      1000,
				Gecos:    "",
				Home:     "/home/rpi",
				Shell:    "/usr/bin/sh",
			},
		},
		Mounts: []mount.MountPoint{
			{
				Target: "/var",
				Chmod:  0o755,
				Source: "/dev/disk/by-label/var",
				Fstype: "ext4",
				Flags:  mount.MountFlagReadWrite | mount.MountFlagRelATime,
				Data:   "discard",
			},
			{
				Target: "/etc",
				Chmod:  0o755,
				Source: "/dev/disk/by-label/etc",
				Fstype: "",
				Flags:  0,
				Data:   "",
			},
		},
		Links: nil,
		Directories: []sr.Directory{
			{
				Overwrite: false,
				Path:      "/root",
				Mode:      0o750,
				UID:       0,
				GID:       0,
			},
			{
				Overwrite: false,
				Path:      "/root/.ssh",
				Mode:      0o700,
				UID:       0,
				GID:       0,
			},
			{
				Overwrite: false,
				Path:      "/home/rpi",
				Mode:      0o750,
				UID:       1000,
				GID:       1000,
			},
			{
				Overwrite: false,
				Path:      "/home/rpi/.ssh",
				Mode:      0o700,
				UID:       1000,
				GID:       1000,
			},
		},
		Files: []sr.File{
			{
				Overwrite: false,
				Filename:  "/root/.ssh/authorized_keys",
				Content:   []byte(strings.Join(config.Users[0].SSHAuthorizedKeys, "\n") + "\n"),
				Mode:      0o600,
				UID:       0,
				GID:       0,
			},
			{
				Overwrite: false,
				Filename:  "/home/rpi/.ssh/authorized_keys",
				Content:   []byte(strings.Join(config.Users[1].SSHAuthorizedKeys, "\n") + "\n"),
				Mode:      0o600,
				UID:       1000,
				GID:       1000,
			},
			{
				Overwrite: true,
				Filename:  "/run/simplek8s/simplek8s.yaml",
				Content:   []byte(config.String()),
				Mode:      0o400,
				UID:       0,
				GID:       0,
			},
		},
	}

	sysroot := &sr.Sysroot{}
	if err := bootstrap.FeedSysrootByBootstrapConfig(sysroot, config); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(sysroot, want) {
		t.Fatalf("got: %v, want: %v", sysroot, want)
	}
}
