package wallet

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
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

var stake = cli.Command{
	Name:  "stake",
	Usage: "Build a stake tx",
	Flags: []cli.Flag{
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.StakePoolFlag,
	},
	Action: func(c *cli.Context) error {
		if c.NumFlags() == 0 {
			cli.ShowSubcommandHelp(c)
			return nil
		}
		if err := CreateStakeTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateStakeTransaction(c *cli.Context) error {
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

	mainAccount, err := account.GetWalletMainAccountData(walletPath)
	p, err := common.HexStringToBytes(mainAccount.ProgramHash)
	if err != nil {
		return err
	}
	programHash, err := common.Uint168FromBytes(p)
	if err != nil {
		return err
	}

	// create inputs
	txInputs, changeOutputs, err := createInputs(mainAccount.Address, totalAmount)
	if err != nil {
		return err
	}

	codeHash := programHash.ToCodeHash()
	depositHash := common.Uint168FromCodeHash(byte(contract.PrefixDPoSV2), codeHash)
	sAddress, err := depositHash.ToAddress()
	sAddressProgramHash, err := common.Uint168FromAddress(sAddress)

	stakePoolStr := c.String("stakepool")
	stakeProgramHash, err := common.Uint168FromAddress(stakePoolStr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// create outputs
	stakeOutput := &outputpayload.ExchangeVotesOutput{
		Version:      0,
		StakeAddress: *sAddressProgramHash,
	}

	var txOutputs = []*common2.Output{
		{
			AssetID:     *account.SystemAssetID,
			Type:        common2.OTStake,
			Value:       *amount,
			OutputLock:  0,
			ProgramHash: *stakeProgramHash,
			Payload:     stakeOutput,
		},
	}

	txOutputs = append(txOutputs, changeOutputs...)

	redeemScript, err := common.HexStringToBytes(mainAccount.RedeemScript)
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
		common2.Stake,
		0,
		&payload.ExchangeVotes{},
		txAttributes,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram},
	)

	OutputTx(0, 1, txn)

	return nil
}
