package wallet

import (
	"fmt"
	"os"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/urfave/cli"
)

var registercr = cli.Command{
	Name:  "registercr",
	Usage: "Build a tx to register cr",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createRegisterCRTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func createRegisterCRTransaction(c *cli.Context) error {
	return createCRInfoCommonTransaction(c, common2.RegisterCR, true)
}
