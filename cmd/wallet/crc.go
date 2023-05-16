package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

func createCRInfoCommonTransaction(c *cli.Context, txType common2.TxType, needOutputAmount bool) error {
	var name string
	var amount *common.Fixed64
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

	outputs := make([]*OutputInfo, 0)
	if needOutputAmount {
		name = cmdcom.TransactionAmountFlag.Name
		amountStr := c.String(name)
		if amountStr == "" {
			return errors.New(fmt.Sprintf("use --%s to specify transfer amount", name))
		}
		amount, err = common.StringToFixed64(amountStr)
		if err != nil {
			return errors.New("invalid transaction amount")
		}

		ct, err := contract.CreateDepositContractByPubKey(acc.PublicKey)
		if err != nil {
			return err
		}
		to, err := ct.ToProgramHash().ToAddress()
		if err != nil {
			return err
		}
		outputs = []*OutputInfo{{to, amount}}
	}

	code, err := contract.CreateStandardRedeemScript(acc.PublicKey)
	if err != nil {
		return errors.New("create standard redeem script failed: " + err.Error())
	}

	newCode := make([]byte, len(code))
	copy(newCode, code)
	didCode := append(newCode[:len(newCode)-1], common.DID)
	ctDID, err := contract.CreateCRIDContractByCode(didCode)
	if err != nil {
		return err
	}

	ctCID, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return err
	}

	name = cmdcom.TransactionNickNameFlag.Name
	nickName := c.String(name)
	if nickName == "" {
		return errors.New(fmt.Sprintf("use --%s to specify nick name", name))
	}

	name = cmdcom.TransactionUrlFlag.Name
	url := c.String(name)
	if url == "" {
		return errors.New(fmt.Sprintf("use --%s to specify url", name))
	}
	locationCode := c.Uint64(cmdcom.TransactionLocationFlag.Name)

	p := &payload.CRInfo{
		Code:      code,
		CID:       *ctCID.ToProgramHash(),
		DID:       *ctDID.ToProgramHash(),
		NickName:  nickName,
		Url:       url,
		Location:  locationCode,
		Signature: nil,
	}

	rpSignBuf := new(bytes.Buffer)
	err = p.SerializeUnsigned(rpSignBuf, payload.CRInfoDIDVersion)
	if err != nil {
		return err
	}

	rpSig, err := acc.Sign(rpSignBuf.Bytes())
	if err != nil {
		return err
	}
	p.Signature = rpSig

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, txType, payload.CRInfoDIDVersion, p, outputs...)
	if err != nil {
		return errors.New("create tx failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func createRegisterCRTransaction(c *cli.Context) error {
	return createCRInfoCommonTransaction(c, common2.RegisterCR, true)
}

func createUpdateCRTransaction(c *cli.Context) error {
	return createCRInfoCommonTransaction(c, common2.UpdateCR, false)
}

func createUnregisterCRTransaction(c *cli.Context) error {
	var name string

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
	outputs := make([]*OutputInfo, 0)

	code, err := contract.CreateStandardRedeemScript(acc.PublicKey)
	if err != nil {
		return errors.New("create standard redeem script failed: " + err.Error())
	}
	ctCID, err := contract.CreateCRIDContractByCode(code)
	if err != nil {
		return err
	}

	p := &payload.UnregisterCR{
		CID: *ctCID.ToProgramHash(),
	}

	rpSignBuf := new(bytes.Buffer)
	err = p.SerializeUnsigned(rpSignBuf, payload.UnregisterCRVersion)
	if err != nil {
		return err
	}

	rpSig, err := acc.Sign(rpSignBuf.Bytes())
	if err != nil {
		return err
	}
	p.Signature = rpSig

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.UnregisterCR,
		payload.UnregisterCRVersion, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func createReturnCRDepositTransaction(c *cli.Context) error {
	return createReturnDepositCommonTransaction(c, common2.ReturnCRDepositCoin)
}

var registercr = cli.Command{
	Name:  "register",
	Usage: "Build a tx to register cr",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createRegisterCRTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var updatecr = cli.Command{
	Name:  "update",
	Usage: "Build a tx to update cr",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createUpdateCRTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var unregistercr = cli.Command{
	Name:  "unregister",
	Usage: "Build a tx to unregister cr",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createUnregisterCRTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var returncrdeposit = cli.Command{
	Name:  "returndeposit",
	Usage: "Build a tx to return deposit coin of CR",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createReturnCRDepositTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var crc = cli.Command{
	Name:  "crc",
	Usage: "Create crc related transactions",
	Flags: []cli.Flag{},
	Subcommands: []cli.Command{
		registercr,
		updatecr,
		unregistercr,
		returncrdeposit,
	},
	Action: func(c *cli.Context) error {
		cli.ShowSubcommandHelp(c)
		return nil
	},
}
