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

var dposV2VoteRenew = cli.Command{
	Name:  "dposv2voterenew",
	Usage: "Build a tx to renew DPoS 2.0 node votes",
	Flags: []cli.Flag{
		cmdcom.TransactionReferKeysFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.AccountWalletFlag,
		cmdcom.CandidatesFlag,
		cmdcom.VotesFlag,
		cmdcom.StakeUntilListFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateDPoSV2VoteRenewTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateDPoSV2VoteRenewTransaction(c *cli.Context) error {

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

	// get refer key of DPoS 2.0 node votes
	referKeysStr := c.String("referkeys")
	if referKeysStr == "" {
		return errors.New("use --referkeys to specify referkey")
	}
	candidates := c.String("candidates")
	if candidates == "" {
		return errors.New("use --candidates to specify candidates information")
	}
	votes := c.String("votes")
	if votes == "" {
		return errors.New("use --votes to specify votes information")
	}
	stakeUntilList := c.String("stakeuntils")
	if stakeUntilList == "" {
		return errors.New("use --stakeuntils to specify stake until information")
	}

	//// get candidate list from file
	//candidatePath := c.String("for")
	//candidateVotesList, err := parseCandidatesAndVotes(candidatePath)
	//if err != nil {
	//	return err
	//}

	contents, err := parseRenewalVotesContent(referKeysStr, candidates, votes, stakeUntilList)
	if err != nil {
		return err
	}
	fmt.Println("contents:", contents)
	pld := payload.Voting{
		RenewalContents: contents,
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
		1,
		&pld,
		txAttributes,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram},
	)

	OutputTx(0, 1, txn)

	// print refer key
	for _, v := range pld.RenewalContents {
		dpld := payload.DetailedVoteInfo{
			TransactionHash: txn.Hash(),
			VoteType:        outputpayload.DposV2,
			Info: []payload.VotesWithLockTime{
				v.VotesInfo,
			},
		}
		fmt.Println("candidate:", common.BytesToHexString(v.VotesInfo.Candidate), "votes:", v.VotesInfo.Votes, "referKey:", dpld.ReferKey())
	}

	return nil
}
