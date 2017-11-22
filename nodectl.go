package main

import (
	"os"
	"sort"

	_ "DNA_POW/cli"
	"DNA_POW/cli/asset"
	"DNA_POW/cli/debug"
	"DNA_POW/cli/dnatst"
	"DNA_POW/cli/info"
	"DNA_POW/cli/mining"
	"DNA_POW/cli/recover"
	"DNA_POW/cli/wallet"

	"github.com/urfave/cli"
)

var Version string

func main() {
	app := cli.NewApp()
	app.Name = "nodectl"
	app.Version = Version
	app.HelpName = "nodectl"
	app.Usage = "command line tool for DNA blockchain"
	app.UsageText = "nodectl [global options] command [command options] [args]"
	app.HideHelp = false
	app.HideVersion = false
	//commands
	app.Commands = []cli.Command{
		*debug.NewCommand(),
		*info.NewCommand(),
		*wallet.NewCommand(),
		*asset.NewCommand(),
		*recover.NewCommand(),
		*mining.NewCommand(),
		*dnatst.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
