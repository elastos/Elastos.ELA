package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

var unregistercr = cli.Command{
	Name:  "unregistercr",
	Usage: "Build a tx to unregister cr",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createUnregisterCRTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func createUnregisterCRTransaction(c *cli.Context) error {
	var name string

	name = cmdcom.TransactionFeeFlag.Name
	feeStr := c.String(name)
	if feeStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer fee", name))
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}

	var acc *account.Account
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	acc = client.GetMainAccount()
	outputs := make([]*OutputInfo, 0)

	code, err := contract.CreateStandardRedeemScript(acc.PublicKey)
	if err != nil {
		return errors.New("create standard redeem script failed: " + err.Error())
	}
	ctCID, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return err
	}

	p := &payload.UnregisterCR{
		CID: *ctCID.ToProgramHash(),
	}

	rpSignBuf := new(bytes.Buffer)
	err = p.SerializeUnsigned(rpSignBuf, payload.UnregisterCRVersion)
	if err != nil {
		return err
	}

	rpSig, err := acc.Sign(rpSignBuf.Bytes())
	if err != nil {
		return err
	}
	p.Signature = rpSig

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.UnregisterCR,
		payload.UnregisterCRVersion, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}
