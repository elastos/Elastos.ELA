package wallet

import (
	"bytes"
	"fmt"
	"errors"
	"os"

	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/account"

	"github.com/urfave/cli"
)

var proposalwithdraw = cli.Command{
	Name:  "proposalwithdraw",
	Usage: "Build a tx to withdraw crc proposal",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.CRCProposalHashFlag,
		cmdcom.CRCProposalStageFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.CRCCommiteeAddrFlag,
		cmdcom.TransactionToFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateCRCProposalWithdrawTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func CreateCRCProposalWithdrawTransaction(c *cli.Context) error {
	walletPath := c.String("wallet")
	if walletPath == "" {
		return errors.New("use --wallet to specify wallet path")
	}
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}
	fmt.Printf("walletPath %s, password %s \n", walletPath, password)

	proposalHashStr := c.String("proposalhash")
	if proposalHashStr == "" {
		return errors.New("use --proposalhash to specify transfer proposalhash")
	}
	CRCCommiteeAddr := c.String("crccommiteeaddr")
	if CRCCommiteeAddr == "" {
		return errors.New("use --crccommiteeaddr to specify from address")
	}
	amountStr := c.String("amount")
	if amountStr == "" {
		return errors.New("use --amount to specify transfer amount")
	}
	amount, err := common.StringToFixed64(amountStr)
	feeStr := c.String("fee")
	if feeStr == "" {
		return errors.New("use --fee to specify transfer fee")
	}
	fee, err := common.StringToFixed64(feeStr)

	fmt.Printf("proposalhash:%s, fee:%s amout:%s CRCCommiteeAddr:%s\n",
		proposalHashStr, feeStr, amountStr, CRCCommiteeAddr)
	*amount -= *fee

	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	var acc *account.Account
	var OwnerPublicKey []byte

	acc = client.GetMainAccount()
	if contract.GetPrefixType(acc.ProgramHash) != contract.PrefixStandard {
		return errors.New("main account is not a standard account")
	}
	OwnerPublicKey, err = acc.PublicKey.EncodePoint(true)
	if err != nil {
		return err
	}
	proposalHash, err2 := common.Uint256FromHexString(proposalHashStr)
	if err2 != nil {
		return err2
	}
	crcProposalWithdraw := &payload.CRCProposalWithdraw{
		ProposalHash:   *proposalHash,
		OwnerPublicKey: OwnerPublicKey,
	}

	signBuf := new(bytes.Buffer)
	crcProposalWithdraw.SerializeUnsigned(signBuf, payload.CRCProposalWithdrawDefault)
	signature, err := acc.Sign(signBuf.Bytes())
	if err != nil {
		return err
	}
	crcProposalWithdraw.Signature = signature

	recipient := c.String("to")
	outputs := make([]*OutputInfo, 0)
	outpusInfo := &OutputInfo{
		Recipient: recipient,
		Amount:    amount,
	}
	outputs = append(outputs, outpusInfo)
	// create outputs
	txOutputs, totalAmount, err := createNormalOutputs(outputs, *fee, uint32(0))
	if err != nil {
		return err
	}
	// create inputs from CRCCommiteeAddr
	txInputs, changeOutputs, err := createInputs(CRCCommiteeAddr, totalAmount)
	if err != nil {
		return err
	}
	txOutputs = append(txOutputs, changeOutputs...)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.CRCProposalWithdraw,
		0,
		crcProposalWithdraw,
		[]*common2.Attribute{},
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{},
	)

	OutputTx(0, 0, txn)

	return nil
}
