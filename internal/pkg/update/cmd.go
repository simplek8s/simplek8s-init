package update

import (
	"flag"
)

const (
	SUBCMD_LIST    string = "list"
	SUBCMD_UPDATE         = "update"
	SUBCMD_CURRENT        = "current"
)

//TODO
func cmdCurrent() error {
	panic("unimplemented")
}

//TODO
func cmdList(providerUrl *string) error {
	panic("unimplemented")
}

//TODO
func cmdUpdate(providerUrl *string, dryRun *bool) error {
	panic("unimplemented")
}

//TODO
func Cmd(args []string) error {
	switch args[1] {
	case SUBCMD_LIST:
		flagList := flag.NewFlagSet("list", flag.ExitOnError)
		flagListProvider := flagList.String("provider", "https://simplek8s.jlsalvador.online/simplek8s/stable", "Use a custom provider for updates")
		flagList.Parse(args[2:])

		if err := cmdList(flagListProvider); err != nil {
			panic(err)
		}
	case SUBCMD_UPDATE:
		flagUpdate := flag.NewFlagSet("update", flag.ExitOnError)
		flagUpdateProvider := flagUpdate.String("provider", "https://simplek8s.jlsalvador.online/simplek8s/stable", "Use a custom provider for updates")
		flagUpdateDryRun := flagUpdate.Bool("dry-run", false, "Dry run")
		flagUpdate.Parse(args[2:])

		if err := cmdUpdate(flagUpdateProvider, flagUpdateDryRun); err != nil {
			panic(err)
		}
	case SUBCMD_CURRENT:
		if err := cmdCurrent(); err != nil {
			panic(err)
		}
	}

	panic("unimplemented")
}
