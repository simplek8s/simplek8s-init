// Copyright 2022 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

package sysroot

import (
	"encoding/json"
	"io/fs"

	"github.com/jlsalvador/simplek8s/pkg/linux/passwd"
)

type Mount struct {
	What    string
	Where   string
	Type    string
	Options string
}

type Link struct {
	Overwrite bool
	Path      string
	Target    string
	Uid       int
	Gid       int
	Hard      bool
}

type Directory struct {
	Overwrite bool
	Path      string
	Mode      fs.FileMode
	Uid       int
	Gid       int
}

type File struct {
	Overwrite bool
	Filename  string
	Content   []byte
	Mode      fs.FileMode
	Uid       int
	Gid       int
}

type Sysroot struct {
	Shadows     []passwd.Shadow
	Groups      []passwd.Group
	Users       []passwd.User
	Mounts      []Mount
	Links       []Link
	Directories []Directory
	Files       []File
}

func (sr *Sysroot) String() string {
	j, _ := json.MarshalIndent(sr, "", "  ")
	return string(j)
}
