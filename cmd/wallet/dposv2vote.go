package wallet

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

var dposV2Vote = cli.Command{
	Name:  "dposv2vote",
	Usage: "Build a tx to vote DPoS 2.0 node",
	Flags: []cli.Flag{
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateDPoSV2VoteTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateDPoSV2VoteTransaction(c *cli.Context) error {

	walletPath := c.String("wallet")

	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}
	fee, err := common.StringToFixed64(feeStr)
	if err != nil {
		return errors.New("invalid transaction fee")
	}

	// get sender from wallet by from address
	from := c.String("from")
	sender, err := getSender(walletPath, from)
	if err != nil {
		return err
	}

	// get candidate list from file
	candidatePath := c.String("for")
	candidateVotesList, err := parseCandidatesAndVotes(candidatePath)
	if err != nil {
		return err
	}
	pld := payload.Voting{
		Contents: []payload.VotesContent{
			{
				VoteType:  outputpayload.DposV2,
				VotesInfo: candidateVotesList,
			},
		},
		RenewalContents: nil,
	}

	// create inputs
	txInputs, changeOutputs, err := createInputs(sender.Address, *fee)
	if err != nil {
		return err
	}
	txOutputs := changeOutputs

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
		common2.Voting,
		0,
		&pld,
		txAttributes,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram},
	)

	OutputTx(0, 1, txn)

	return nil
}
