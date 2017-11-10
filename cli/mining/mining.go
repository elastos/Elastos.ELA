package mining

import (
	. "DNA_POW/cli/common"
	"DNA_POW/net/httpjsonrpc"
	"errors"

	"github.com/urfave/cli"
)

func miningAction(c *cli.Context) (err error) {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	toggle := c.Bool("toggle")
	if toggle {
		control := c.String("control")
		var isMining bool
		if control == "start" || control == "START" {
			isMining = true
		} else if control == "stop" || control == "STOP" {
			isMining = false
		} else {
			return errors.New("argument 'control' is must be 'start' or 'stop'")
		}
		resp, _ := httpjsonrpc.Call(Address(), "togglecpumining", 0, []interface{}{isMining})
		FormatOutput(resp)
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "mining",
		Usage:       "toggle cpu mining.",
		Description: "With nodectl test, you could toggle cpu mining.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "toggle, t",
				Usage: "toggle mining",
			},
			cli.StringFlag{
				Name:  "control, c",
				Usage: "control mining",
			},
		},
		Action: miningAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "mining")
			return cli.NewExitError("", 1)
		},
	}
}
