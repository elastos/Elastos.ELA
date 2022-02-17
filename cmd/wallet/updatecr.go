package wallet

import (
	"fmt"
	"os"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/urfave/cli"
)

var updatecr = cli.Command{
	Name:  "updatecr",
	Usage: "Build a tx to update cr",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createUpdateCRTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func createUpdateCRTransaction(c *cli.Context) error {
	return createCRInfoCommonTransaction(c, common2.UpdateCR, false)
}
