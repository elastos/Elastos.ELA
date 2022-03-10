package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
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
	amount := c.Int64("claimamount")
	if amount == 0 {
		return errors.New("must specify claimamount flag")
	}
	walletPath := c.String("wallet")

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
	if contract.GetPrefixType(acc.ProgramHash) != contract.PrefixStandard {
		return errors.New("main account is not a standard account")
	}
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
	apPayload := &payload.DposV2ClaimReward{
		Amount: common.Fixed64(amount),
	}

	if err = apPayload.SerializeUnsigned(buf, payload.ActivateProducerVersion); err != nil {
		return err
	}
	signature, err := acc.Sign(buf.Bytes())
	if err != nil {
		return err
	}
	apPayload.Signature = signature

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
