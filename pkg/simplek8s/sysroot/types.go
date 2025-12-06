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

package sysroot

import (
	"encoding/json"
	"io/fs"

	"github.com/simplek8s/simplek8s-init/pkg/linux/mount"
	"github.com/simplek8s/simplek8s-init/pkg/linux/passwd"
)

type Link struct {
	Overwrite bool
	Path      string
	Target    string
	UID       int
	GID       int
	Hard      bool
}

type Directory struct {
	Overwrite bool
	Path      string
	Mode      fs.FileMode
	UID       int
	GID       int
}

type File struct {
	Overwrite bool
	Filename  string
	Content   []byte
	Mode      fs.FileMode
	UID       int
	GID       int
}

type Sysroot struct {
	Shadows     []passwd.Shadow
	Groups      []passwd.Group
	Users       []passwd.User
	Mounts      []mount.MountPoint
	Links       []Link
	Directories []Directory
	Files       []File
}

func (sr *Sysroot) String() string {
	j, _ := json.MarshalIndent(sr, "", "  ")
	return string(j)
}
