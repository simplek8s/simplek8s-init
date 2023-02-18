package templates

import "embed"

type TmplDataSystemdUnitMount struct {
	DefaultDependencies bool
	BindsTo             []string
	Conflicts           []string
	Before              []string
	After               []string
	Requires            []string
	Where               string
	What                string
	Type                string
	Options             string
}

type TmplDataSystemdUnitService struct {
	DefaultDependencies bool
	Description         string
	BindsTo             []string
	Conflicts           []string
	Before              []string
	After               []string
	Requires            []string
	ConditionPathExists []string
	Type                string
	Restart             string
	RemainAfterExit     bool
	ExecStart           []string
	ExecStop            []string
}

//go:embed assets/*
var Templates embed.FS
