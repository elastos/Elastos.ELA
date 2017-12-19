package asset

import (
	"fmt"
	"os"

	. "ELA/cli/common"
	"ELA/net/httpjsonrpc"

	"github.com/urfave/cli"
)

func assetAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}
	if !c.Bool("transfer") {
		fmt.Println("missing flag [--transfer]")
		return nil
	}
	asset := c.String("asset")
	if asset == "" {
		fmt.Println("missing flag [--asset]")
		return nil
	}
	address := c.String("to")
	if address == "" {
		fmt.Println("missing flag [--to]")
		return nil
	}
	value := c.String("value")
	if value == "" {
		fmt.Println("asset amount is required with [--value]")
		return nil
	}
	fee := c.String("fee")
	if fee == "" {
		fmt.Println("transaction fee is required with [--fee]")
		return nil
	}
	resp, err := httpjsonrpc.Call(Address(), "sendtransaction", 0, []interface{}{asset, address, value,fee})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	FormatOutput(resp)

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "asset",
		Usage:       "asset registration, issuance and transfer",
		Description: "With nodectl asset, you could control assert through transaction.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "transfer, t",
				Usage: "transfer asset",
			},
			cli.StringFlag{
				Name:  "asset, a",
				Usage: "uniq id for asset",
			},
			cli.StringFlag{
				Name:  "to",
				Usage: "asset to whom",
			},
			cli.StringFlag{
				Name:  "value, v",
				Usage: "asset amount",
				Value: "",
			},
			cli.StringFlag{
				Name:  "fee, f",
				Usage: "transaction fee",
				Value: "",
			},
		},
		Action: assetAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "asset")
			return cli.NewExitError("", 1)
		},
	}
}
