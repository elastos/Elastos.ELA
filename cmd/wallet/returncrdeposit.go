package wallet

import (
	"fmt"
	"os"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/urfave/cli"
)

var returncrdeposit = cli.Command{
	Name:  "returncrdeposit",
	Usage: "Build a tx to return deposit coin of CR",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createReturnCRDepositTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func createReturnCRDepositTransaction(c *cli.Context) error {
	return createReturnDepositCommonTransaction(c, common2.ReturnCRDepositCoin)
}
