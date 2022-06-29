package templates

import "embed"

type TmplDataSystemdUnitMount struct {
	DefaultDependencies bool
	Before              []string
	After               []string
	Where               string
	What                string
	Type                string
	Options             string
}

type TmplDataSystemdUnitService struct {
	Description         string
	Before              []string
	After               []string
	ConditionPathExists []string
	Type                string
	ExecStart           []string
}

//go:embed assets/*
var Templates embed.FS
