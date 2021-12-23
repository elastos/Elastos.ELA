package wallet

import (
	"fmt"
	"errors"
	"math/rand"
	"os"
	"strconv"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

var vote = cli.Command{
	Name:  "vote",
	Usage: "Build a tx to vote for candidates using ELA",
	Flags: []cli.Flag{
		cmdcom.TransactionForFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFromFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
	},
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowSubcommandHelp(c)
			return nil
		}
		if err := CreateVoteTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateVoteTransaction(c *cli.Context) error {

	walletPath := c.String("wallet")

	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	// calculate total amount
	amountStr := c.String("amount")
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}
	amount, err := common.StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}
	totalAmount := *fee + *amount

	// get sender from wallet by from address
	from := c.String("from")
	sender, err := getSender(walletPath, from)
	if err != nil {
		return err
	}

	// get candidate list from file
	candidatePath := c.String("for")
	candidateList, err := parseCandidates(candidatePath)
	if err != nil {
		return err
	}

	// create outputs
	txOutputs, err := createVoteOutputs(&OutputInfo{
		Recipient: sender.Address,
		Amount:    amount,
	}, candidateList)
	if err != nil {
		return err
	}

	// create inputs
	txInputs, changeOutputs, err := createInputs(sender.Address, totalAmount)
	if err != nil {
		return err
	}
	txOutputs = append(txOutputs, changeOutputs...)

	redeemScript, err := common.HexStringToBytes(sender.RedeemScript)
	if err != nil {
		return err
	}

	// create attributes
	txAttr := common2.NewAttribute(common2.Nonce, []byte(strconv.FormatInt(rand.Int63(), 10)))
	txAttributes := make([]*common2.Attribute, 0)
	txAttributes = append(txAttributes, &txAttr)

	// create program
	var txProgram = &pg.Program{
		Code:      redeemScript,
		Parameter: nil,
	}

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		txAttributes,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram},
	)

	OutputTx(0, 1, txn)

	return nil
}
