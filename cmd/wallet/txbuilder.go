// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

type OutputInfo struct {
	Recipient string
	Amount    *common.Fixed64
}

type CrossChainOutput struct {
	Recipient         string
	Amount            *common.Fixed64
	CrossChainAddress string
}

func CreateTransaction(c *cli.Context) error {
	walletPath := c.String("wallet")

	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}

	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	from := c.String("from")

	outputs := make([]*OutputInfo, 0)
	to := c.String("to")
	amountStr := c.String("amount")
	toMany := c.String("tomany")
	if toMany != "" {
		if to != "" {
			return errors.New("'--to' cannot be specified when specify '--tomany' option")
		}
		if amountStr != "" {
			return errors.New("'--amount' cannot be specified when specify '--tomany' option")
		}
		outputs, err = parseMultiOutput(toMany)
		if err != nil {
			return err
		}
	} else {
		if amountStr == "" {
			return errors.New("use --amount to specify transfer amount")
		}
		amount, err := common.StringToFixed64(amountStr)
		if err != nil {
			return errors.New("invalid transaction amount")
		}
		if to == "" {
			return errors.New("use --to to specify recipient")
		}
		outputs = []*OutputInfo{{to, amount}}
	}

	outputLockStr := c.String("outputlock")
	outputLock := uint64(0)
	if outputLockStr != "" {
		outputLock, err = strconv.ParseUint(outputLockStr, 10, 32)
		if err != nil {
			return errors.New("invalid output lock height")
		}
	}

	txLockStr := c.String("txlock")
	txLock := uint64(0)
	if txLockStr != "" {
		txLock, err = strconv.ParseUint(txLockStr, 10, 32)
		if err != nil {
			return errors.New("invalid transaction lock height")
		}
	}

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, from, *fee, uint32(outputLock),
		uint32(txLock), common2.TransferAsset, 0, &payload.TransferAsset{}, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func getSender(walletPath string, from string) (*account.AccountData, error) {
	var sender *account.AccountData
	mainAccount, err := account.GetWalletMainAccountData(walletPath)
	if err != nil {
		return nil, err
	}

	if from == "" {
		from = mainAccount.Address
		sender = mainAccount
	} else {
		storeAccounts, err := account.GetWalletAccountData(walletPath)
		if err != nil {
			return nil, err
		}
		for _, acc := range storeAccounts {
			if from == acc.Address {
				sender = &acc
				break
			}
		}
		if sender == nil {
			return nil, errors.New(from + " is not local account")
		}
	}

	return sender, nil
}

func createInputs(fromAddr string, totalAmount common.Fixed64) ([]*common2.Input,
	[]*common2.Output, error) {
	UTXOs, err := getUTXOsByAmount(fromAddr, totalAmount)
	if err != nil {
		return nil, nil, err
	}

	var txInputs []*common2.Input
	var changeOutputs []*common2.Output
	for _, utxo := range UTXOs {
		txIDReverse, _ := hex.DecodeString(utxo.TxID)
		txID, _ := common.Uint256FromBytes(common.BytesReverse(txIDReverse))
		sequence := math.MaxUint32
		if utxo.OutputLock > 0 {
			sequence = math.MaxUint32 - 1
		}
		input := &common2.Input{
			Previous: common2.OutPoint{
				TxID:  *txID,
				Index: utxo.VOut,
			},
			Sequence: uint32(sequence),
		}
		txInputs = append(txInputs, input)
		amount, err := common.StringToFixed64(utxo.Amount)
		if err != nil {
			return nil, nil, err
		}
		programHash, err := common.Uint168FromAddress(fromAddr)
		if err != nil {
			return nil, nil, err
		}
		if *amount < totalAmount {
			totalAmount -= *amount
		} else if *amount == totalAmount {
			totalAmount = 0
			break
		} else if *amount > totalAmount {
			change := &common2.Output{
				AssetID:     *account.SystemAssetID,
				Value:       *amount - totalAmount,
				OutputLock:  uint32(0),
				ProgramHash: *programHash,
				Type:        common2.OTNone,
				Payload:     &outputpayload.DefaultOutput{},
			}
			changeOutputs = append(changeOutputs, change)
			totalAmount = 0
			break
		}
	}
	if totalAmount > 0 {
		return nil, nil, errors.New("[Wallet], Available token is not enough")
	}

	return txInputs, changeOutputs, nil
}

func createNormalOutputs(outputs []*OutputInfo, fee common.Fixed64, lockedUntil uint32) ([]*common2.Output, common.Fixed64, error) {
	var totalAmount = common.Fixed64(0) // The total amount will be spend
	var txOutputs []*common2.Output     // The outputs in transaction
	totalAmount += fee                  // Add transaction fee

	for _, output := range outputs {
		recipient, err := common.Uint168FromAddress(output.Recipient)
		if err != nil {
			return nil, 0, errors.New(fmt.Sprint("invalid receiver address: ", output.Recipient, ", error: ", err))
		}

		txOutput := &common2.Output{
			AssetID:     *account.SystemAssetID,
			ProgramHash: *recipient,
			Value:       *output.Amount,
			OutputLock:  lockedUntil,
			Type:        common2.OTNone,
			Payload:     &outputpayload.DefaultOutput{},
		}
		totalAmount += *output.Amount
		txOutputs = append(txOutputs, txOutput)
	}

	if totalAmount <= 0 {
		return nil, 0, errors.New("outputs total amount plus fee should not be less than or equal to 0")
	}

	return txOutputs, totalAmount, nil
}

func createVoteOutputs(output *OutputInfo, candidateList []string) ([]*common2.Output, error) {
	var txOutputs []*common2.Output
	recipient, err := common.Uint168FromAddress(output.Recipient)
	if err != nil {
		return nil, errors.New(fmt.Sprint("invalid receiver address: ", output.Recipient, ", error: ", err))
	}

	// create vote output payload
	var cv []outputpayload.CandidateVotes
	for _, candidateHex := range candidateList {
		candidateBytes, err := common.HexStringToBytes(candidateHex)
		if err != nil {
			return nil, err
		}
		cv = append(cv, outputpayload.CandidateVotes{
			Candidate: candidateBytes,
		})
	}
	voteContent := outputpayload.VoteContent{
		VoteType:       outputpayload.Delegate,
		CandidateVotes: cv,
	}
	voteOutput := outputpayload.VoteOutput{
		Version: 0,
		Contents: []outputpayload.VoteContent{
			voteContent,
		},
	}

	txOutput := &common2.Output{
		AssetID:     *account.SystemAssetID,
		ProgramHash: *recipient,
		Value:       *output.Amount,
		OutputLock:  0,
		Type:        common2.OTVote,
		Payload:     &voteOutput,
	}
	txOutputs = append(txOutputs, txOutput)

	return txOutputs, nil
}

func createTransaction(walletPath string, from string, fee common.Fixed64, outputLock uint32, txLock uint32,
	txType common2.TxType, payloadVersion byte, payload interfaces.Payload,
	outputs ...*OutputInfo) (interfaces.Transaction, error) {

	// get sender in wallet by from address
	sender, err := getSender(walletPath, from)
	if err != nil {
		return nil, err
	}

	// create outputs
	txOutputs, totalAmount, err := createNormalOutputs(outputs, fee, outputLock)
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
		txType,
		payloadVersion,
		payload,
		txAttributes,
		txInputs,
		txOutputs,
		txLock,
		[]*pg.Program{txProgram},
	), nil
}

func createReturnDepositCommonTransaction(c *cli.Context, txType common2.TxType) error {
	var name string
	var err error

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

	name = cmdcom.TransactionAmountFlag.Name
	amountStr := c.String(name)
	if amountStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer amount", name))
	}
	amount, err := common.StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}
	*amount -= *fee

	outputs := make([]*OutputInfo, 0)
	outputs = []*OutputInfo{{acc.Address, amount}}

	p := &payload.ReturnDepositCoin{}

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0,
		txType, payload.ReturnDepositCoinVersion, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}
