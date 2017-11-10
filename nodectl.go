package main

import (
	"os"
	"sort"

	_ "DNA_POW/cli"
	"DNA_POW/cli/asset"
	"DNA_POW/cli/bookkeeper"
	. "DNA_POW/cli/common"
	"DNA_POW/cli/data"
	"DNA_POW/cli/debug"
	"DNA_POW/cli/info"
	"DNA_POW/cli/mining"
	"DNA_POW/cli/privpayload"
	"DNA_POW/cli/recover"
	"DNA_POW/cli/test"
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
	//global options
	app.Flags = []cli.Flag{
		NewIpFlag(),
		NewPortFlag(),
	}
	//commands
	app.Commands = []cli.Command{
		*debug.NewCommand(),
		*info.NewCommand(),
		*test.NewCommand(),
		*wallet.NewCommand(),
		*asset.NewCommand(),
		*privpayload.NewCommand(),
		*data.NewCommand(),
		*bookkeeper.NewCommand(),
		*recover.NewCommand(),
		*mining.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
