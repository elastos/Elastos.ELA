package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

var unstake = cli.Command{
	Name:  "unstake",
	Usage: "Build a tx to unstake",
	Flags: []cli.Flag{
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.PayloadVersionFlag,
		cmdcom.TransactionToFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateUnstakeTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateUnstakeTransaction(c *cli.Context) error {
	amount := c.Int64(cmdcom.TransactionAmountFlag.Name)
	if amount == 0 {
		return errors.New("must specify amount flag")
	}
	walletPath := c.String("wallet")

	toAddr := c.String(cmdcom.TransactionToFlag.Name)
	if toAddr == "" {
		return errors.New("use --to to specify recipient address")
	}
	to, err := common.Uint168FromAddress(toAddr)
	if err != nil {
		return err
	}

	feeStr := c.String(cmdcom.TransactionFeeFlag.Name)
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	payloadVersion := c.Int64(cmdcom.PayloadVersionFlag.Name)

	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	acc := client.GetMainAccount()
	mainAccount, err := account.GetWalletMainAccountData(walletPath)
	if err != nil {
		return err
	}

	p := &payload.Unstake{
		ToAddr: *to,
		// unstake value
		Value: common.Fixed64(amount),
	}

	if byte(payloadVersion) == payload.UnstakeVersionV0 {
		redeemScript, err := common.HexStringToBytes(mainAccount.RedeemScript)
		if err != nil {
			fmt.Println("err", err)
			return err
		}
		p.Code = redeemScript

		rpSignBuf := new(bytes.Buffer)
		err = p.SerializeUnsigned(rpSignBuf, byte(payloadVersion))
		if err != nil {
			return err
		}

		rpSig, err := acc.Sign(rpSignBuf.Bytes())
		if err != nil {
			return err
		}
		p.Signature = rpSig
	}

	outputs := make([]*OutputInfo, 0)
	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.Unstake,
		byte(payloadVersion), p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)
	return nil
}
