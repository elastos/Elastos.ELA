package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

var dpossv2claimreward = cli.Command{
	Name:  "dposv2claimreward",
	Usage: "Build a tx to claim dposV2 reward",
	Flags: []cli.Flag{
		cmdcom.TransactionClaimAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.PayloadVersionFlag,
		cmdcom.TransactionToFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateDposV2ClaimRewardTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateDposV2ClaimRewardTransaction(c *cli.Context) error {
	amountStr := c.String("claimamount")
	if amountStr == "" {
		return errors.New("use --claimamount to specify claim amount")
	}
	amount, err := common.StringToFixed64(amountStr)
	if err != nil {
		return errors.New("")
	}

	walletPath := c.String("wallet")

	toAddr := c.String("to")
	if toAddr == "" {
		return errors.New("use --to to specify recipient address")
	}
	to, err := common.Uint168FromAddress(toAddr)
	if err != nil {
		return err
	}

	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	zero, err := common.StringToFixed64("0")
	if err != nil {
		return errors.New("invalid zero")
	}

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

	outputs := make([]*OutputInfo, 0)
	outputs = append(outputs, &OutputInfo{
		Recipient: mainAccount.Address,
		Amount:    zero,
	})

	txOutputs, _, err := createNormalOutputs(outputs, *fee, 0)
	if err != nil {
		return err
	}

	txInputs, changeOutputs, err := createInputs(mainAccount.Address, *fee)
	if err != nil {
		return err
	}
	txOutputs = append(txOutputs, changeOutputs...)

	redeemScript, err := common.HexStringToBytes(mainAccount.RedeemScript)
	if err != nil {
		return err
	}

	// create program
	var txProgram = &pg.Program{
		Code:      redeemScript,
		Parameter: nil,
	}

	buf := new(bytes.Buffer)
	apPayload := &payload.DPoSV2ClaimReward{
		ToAddr: *to,
		Value:  *amount,
	}

	payloadVersion := c.Int64(cmdcom.PayloadVersionFlag.Name)
	if byte(payloadVersion) == payload.DposV2ClaimRewardVersionV0 {
		apPayload.Code = redeemScript
		if err = apPayload.SerializeUnsigned(buf, byte(payloadVersion)); err != nil {
			return err
		}
		signature, err := acc.Sign(buf.Bytes())
		if err != nil {
			return err
		}
		apPayload.Signature = signature
	}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.DposV2ClaimReward,
		0,
		apPayload,
		nil,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram})

	OutputTx(0, 1, txn)

	return nil
}
