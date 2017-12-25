package main

import (
	"os"
	"sort"

	_ "Elastos.ELA/cli"
	"Elastos.ELA/cli/asset"
	"Elastos.ELA/cli/debug"
	"Elastos.ELA/cli/elatst"
	"Elastos.ELA/cli/info"
	"Elastos.ELA/cli/mining"
	"Elastos.ELA/cli/multisig"
	"Elastos.ELA/cli/recover"
	"Elastos.ELA/cli/wallet"

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
		*elatst.NewCommand(),
		*multisig.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
