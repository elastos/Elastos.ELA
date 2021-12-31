package wallet

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/elastos/Elastos.ELA/account"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"

	"github.com/urfave/cli"
)

var returndeposit = cli.Command {
	Name:  "returndeposit",
	Usage: "Build a tx to return deposit coin of producer",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createReturnProducerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

func createReturnProducerTransaction(c *cli.Context) error {
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
	var mainPublicKey []byte
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	acc = client.GetMainAccount()
	mainPublicKey, err = acc.PublicKey.EncodePoint(true)
	if err != nil {
		return err
	}
	programHash, err := contract.PublicKeyToDepositProgramHash(mainPublicKey)
	if (err != nil) {
		return err
	}
	from, err := programHash.ToAddress()

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

	// create outputs
	txOutputs, totalAmount, err := createNormalOutputs(outputs, *fee, 0)
	if err != nil {
		return err
	}

	// create inputs
	txInputs, changeOutputs, err := createInputs(from, totalAmount)
	if err != nil {
		return err
	}
	txOutputs = append(txOutputs, changeOutputs...)

	redeemScript, err := contract.CreateStandardRedeemScript(acc.PublicKey)
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
		common2.ReturnDepositCoin,
		payload.ReturnDepositCoinVersion,
		p,
		txAttributes,
		txInputs,
		txOutputs,
		0,
		[]*pg.Program{txProgram},
	)

	OutputTx(0, 1, txn)

	return nil
}
