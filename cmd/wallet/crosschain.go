package wallet

import (
	"fmt"
	"errors"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"math/rand"
	"os"
	"strconv"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"

	"github.com/urfave/cli"
)

var crosschain = cli.Command{
	Name:  "crosschain",
	Usage: "Build a cross chain tx",
	Flags: []cli.Flag{
		cmdcom.TransactionSAddressFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFromFlag,
		cmdcom.TransactionToFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
	},
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowSubcommandHelp(c)
			return nil
		}
		if err := CreateCrossChainTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateCrossChainTransaction(c *cli.Context) error {
	walletPath := c.String("wallet")

	from := c.String("from")
	to := c.String("to")
	if to == "" {
		return errors.New("use --to to specify a side chain address which want to recharge")
	}
	sAddress := c.String("saddress")
	if sAddress == "" {
		return errors.New("use --saddress to specify a locked address of side chain")
	}

	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	amountStr := c.String("amount")
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}
	amount, err := common.StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

	txn, err := createCrossChainTransaction(walletPath, from, *fee, 0, &CrossChainOutput{
		Recipient:         sAddress,
		Amount:            amount,
		CrossChainAddress: to,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	OutputTx(0, 1, txn)

	return nil
}

func createCrossChainTransaction(walletPath string, from string, fee common.Fixed64, lockedUntil uint32,
	crossChainOutputs ...*CrossChainOutput) (interfaces.Transaction, error) {

	// check output
	if len(crossChainOutputs) == 0 {
		return nil, errors.New("invalid transaction target")
	}

	outputs := make([]*OutputInfo, 0)
	perAccountFee := fee / common.Fixed64(len(crossChainOutputs))

	// create payload
	payload := &payload.TransferCrossChainAsset{}
	for index, output := range crossChainOutputs {
		payload.CrossChainAddresses = append(payload.CrossChainAddresses, output.CrossChainAddress)
		payload.OutputIndexes = append(payload.OutputIndexes, uint64(index))
		payload.CrossChainAmounts = append(payload.CrossChainAmounts, *output.Amount-perAccountFee)
		outputs = append(outputs, &OutputInfo{
			Recipient: output.Recipient,
			Amount:    output.Amount,
		})
	}

	// create outputs
	txOutputs, totalAmount, err := createNormalOutputs(outputs, fee, lockedUntil)
	if err != nil {
		return nil, err
	}

	// get sender in wallet by from address
	sender, err := getSender(walletPath, from)
	if err != nil {
		return nil, err
	}

	// create inputs
	txInputs, changeOutputs, err := createInputs(sender.Address, totalAmount)
	if err != nil {
		return nil, err
	}
	txOutputs = append(txOutputs, changeOutputs...)

	redeemScript, err := common.HexStringToBytes(sender.RedeemScript)
	if err != nil {
		return nil, err
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

	return functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferCrossChainAsset,
		0,
		payload,
		txAttributes,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram},
	), nil
}
