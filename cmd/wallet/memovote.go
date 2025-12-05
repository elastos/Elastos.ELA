package wallet

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

var memoVote = cli.Command{
	Name:  "memovote",
	Usage: "Build a tx to initate poll by memo",
	Flags: []cli.Flag{
		cmdcom.TransactionFromFlag,
		cmdcom.TransactionToFlag,
		cmdcom.TransactionToManyFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionOutputLockFlag,
		cmdcom.TransactionTxLockFlag,
		cmdcom.TransactionMemoType,
		cmdcom.TransactionMemoVotingTime,
		cmdcom.TransctionMemoDescription,
		cmdcom.TransactionMemoChoices,
		cmdcom.TransactionMemoUrl,
		cmdcom.AccountWalletFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateMemoInitatePollTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

type InitateVoting struct {
	Type        byte
	EndTime     uint64
	Description string
	ChoiceCount uint32
	Choices     []string
	Url         string

	// not from memo
	StartTime uint64
	ID        common.Uint256
}

func (v *InitateVoting) Serialize(w io.Writer) error {
	err := v.SerializeUnsigned(w)
	if err != nil {
		return err
	}
	err = common.WriteUint64(w, v.StartTime)
	if err != nil {
		return err
	}
	err = v.ID.Serialize(w)
	if err != nil {
		return err
	}
	return nil
}

func (v *InitateVoting) SerializeUnsigned(w io.Writer) error {
	if _, err := w.Write([]byte{v.Type}); err != nil {
		return err
	}
	err := common.WriteUint64(w, v.EndTime)
	if err != nil {
		return err
	}
	err = common.WriteVarString(w, v.Description)
	if err != nil {
		return err
	}
	err = common.WriteUint32(w, v.ChoiceCount)
	if err != nil {
		return err
	}
	for _, option := range v.Choices {
		err = common.WriteVarString(w, option)
		if err != nil {
			return err
		}
	}
	err = common.WriteVarString(w, v.Url)
	if err != nil {
		return err
	}
	return nil
}

func (v *InitateVoting) Deserialize(r io.Reader) error {
	err := v.DeserializeUnsigned(r)
	if err != nil {
		return err
	}
	v.StartTime, err = common.ReadUint64(r)
	if err != nil {
		return err
	}
	err = v.ID.Deserialize(r)
	if err != nil {
		return err
	}
	return nil
}

func (v *InitateVoting) DeserializeUnsigned(r io.Reader) error {
	vType, err := common.ReadBytes(r, 1)
	if err != nil {
		return err
	}
	v.Type = vType[0]
	v.EndTime, err = common.ReadUint64(r)
	if err != nil {
		return err
	}
	v.Description, err = common.ReadVarString(r)
	if err != nil {
		return err
	}
	v.ChoiceCount, err = common.ReadUint32(r)
	if err != nil {
		return err
	}
	for i := uint32(0); i < v.ChoiceCount; i++ {
		option, err := common.ReadVarString(r)
		if err != nil {
			return err
		}
		v.Choices = append(v.Choices, option)
	}
	v.Url, err = common.ReadVarString(r)
	if err != nil {
		return err
	}
	return nil
}

func CreateMemoInitatePollTransaction(c *cli.Context) error {
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

	typeStr := c.String("type")
	if typeStr == "" {
		return errors.New("use --type to specify poll type, 0 for testing")
	}
	typeInt, err := strconv.ParseUint(typeStr, 10, 64)
	if err != nil {
		return errors.New("invalid poll type")
	}
	if typeInt > 255 || typeInt < 0 {
		return errors.New("poll type must be less than 255 and greater than 0")
	}

	votingTime := c.String("votingtime")
	if votingTime == "" {
		return errors.New("use --votingtime to specify poll voting time in seconds")
	}
	votingTimeInt, err := strconv.ParseUint(votingTime, 10, 64)
	if err != nil {
		return errors.New("invalid voting time")
	}
	endTime := time.Now().Unix() + int64(votingTimeInt)

	description := c.String("description")
	if description == "" {
		return errors.New("use --description to specify poll description")
	}

	choices := c.String("choices")
	if choices == "" {
		return errors.New("use --choices to specify poll choices")
	}
	choicesList := strings.Split(choices, ",")

	pollUrl := c.String("url")
	if pollUrl == "" {
		return errors.New("use --url to specify poll url")
	}

	fmt.Println("type: ", typeInt)
	fmt.Println("endTime: ", endTime)
	fmt.Println("description: ", description)
	fmt.Println("choicesList: ", choicesList)
	fmt.Println("pollUrl: ", pollUrl)
	initatePollMemo := InitateVoting{
		Type:        byte(typeInt),
		EndTime:     uint64(endTime),
		Description: description,
		ChoiceCount: uint32(len(choicesList)),
		Choices:     choicesList,
		Url:         pollUrl,
	}
	var initatePollMemoBytes bytes.Buffer
	// write inittate flag first
	initPollFlag := "pollinit"
	initatePollMemoBytes.Write([]byte(initPollFlag))
	initatePollMemo.SerializeUnsigned(&initatePollMemoBytes)

	// test deserialize
	var initatePollMemo2 InitateVoting
	initatePollMemo2.DeserializeUnsigned(bytes.NewReader(initatePollMemoBytes.Bytes()[8:]))
	fmt.Println("memo: ", hex.EncodeToString(initatePollMemoBytes.Bytes()))
	fmt.Println("memo: ", initatePollMemo2)

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
	txn, err = createTransactionWithMemo(walletPath, from, *fee, uint32(outputLock),
		uint32(txLock), common2.TransferAsset, 0, &payload.TransferAsset{}, initatePollMemoBytes.Bytes(), outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}
	fmt.Println(txn.String())
	OutputTx(0, 1, txn)

	return nil
}
