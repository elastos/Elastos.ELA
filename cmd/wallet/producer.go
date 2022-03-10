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
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/urfave/cli"
)

func createProducerInfoCommonTransaction(c *cli.Context, txType common2.TxType, needOutputAmount bool) error {
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
	var ownerPublicKey []byte
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	acc = client.GetMainAccount()
	ownerPublicKey, err = acc.PublicKey.EncodePoint(true)
	if err != nil {
		return err
	}

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

		programHash, err := contract.PublicKeyToDepositProgramHash(ownerPublicKey)
		if err != nil {
			return err
		}
		to, err := programHash.ToAddress()
		if err != nil {
			return err
		}
		outputs = []*OutputInfo{{to, amount}}
	}

	var nodePublicKey []byte
	name = cmdcom.TransactionNodePublicKeyFlag.Name
	nodePublicKeyStr := c.String(name)
	if nodePublicKeyStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify node pubkey", name))
	}
	nodePublicKey, err = common.HexStringToBytes(nodePublicKeyStr)
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

	name = cmdcom.TransactionNetAddressFlag.Name
	netAddress := c.String(name)
	if netAddress == "" {
		return errors.New(fmt.Sprintf("use --%s to specify ip address of producer", name))
	}

	name = cmdcom.TransactionStakeUntilFlag.Name
	stakeUntil := c.Uint(name)
	if stakeUntil == 0 {
		return errors.New(fmt.Sprintf("use --%s to specify until block height", name))
	}

	p := &payload.ProducerInfo{
		OwnerPublicKey: ownerPublicKey,
		NodePublicKey:  nodePublicKey,
		NickName:       nickName,
		Url:            url,
		Location:       locationCode,
		NetAddress:     netAddress,
		StakeUntil:     uint32(stakeUntil),
	}

	rpSignBuf := new(bytes.Buffer)
	err = p.SerializeUnsigned(rpSignBuf, payload.ProducerInfoDposV2Version)
	if err != nil {
		return err
	}

	rpSig, err := acc.Sign(rpSignBuf.Bytes())
	if err != nil {
		return err
	}
	p.Signature = rpSig

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, txType,
		payload.ProducerInfoDposV2Version, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func createRegisterProducerTransaction(c *cli.Context) error {
	return createProducerInfoCommonTransaction(c, common2.RegisterProducer, true)
}

func createUpdateProducerTransaction(c *cli.Context) error {
	return createProducerInfoCommonTransaction(c, common2.UpdateProducer, false)
}

func createUnregisterProducerTransaction(c *cli.Context) error {
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
	var ownerPublicKey []byte
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	acc = client.GetMainAccount()
	ownerPublicKey, err = acc.PublicKey.EncodePoint(true)
	if err != nil {
		return err
	}
	outputs := make([]*OutputInfo, 0)

	p := &payload.ProcessProducer{
		OwnerPublicKey: ownerPublicKey,
	}

	rpSignBuf := new(bytes.Buffer)
	err = p.SerializeUnsigned(rpSignBuf, payload.ProcessProducerVersion)
	if err != nil {
		return err
	}

	rpSig, err := acc.Sign(rpSignBuf.Bytes())
	if err != nil {
		return err
	}
	p.Signature = rpSig

	var txn interfaces.Transaction
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.CancelProducer,
		payload.ProcessProducerVersion, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}

func createReturnProducerDepositTransaction(c *cli.Context) error {
	return createReturnDepositCommonTransaction(c, common2.ReturnDepositCoin)
}

func CreateActivateProducerTransaction(c *cli.Context) error {
	walletPath := c.String("wallet")
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}

	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}

	var acc *account.Account
	var nodePublicKey []byte

	nodePublicKeyStr := c.String("nodepublickey")
	if nodePublicKeyStr != "" {
		nodePublicKey, err = common.HexStringToBytes(nodePublicKeyStr)
		if err != nil {
			return err
		}
		codeHash, err := contract.PublicKeyToStandardCodeHash(nodePublicKey)
		if err != nil {
			return err
		}
		acc = client.GetAccountByCodeHash(*codeHash)
		if acc == nil {
			return errors.New("no available account in wallet")
		}
	} else {
		acc = client.GetMainAccount()
		if contract.GetPrefixType(acc.ProgramHash) != contract.PrefixStandard {
			return errors.New("main account is not a standard account")
		}
		nodePublicKey, err = acc.PublicKey.EncodePoint(true)
		if err != nil {
			return err
		}
	}

	buf := new(bytes.Buffer)
	apPayload := &payload.ActivateProducer{
		NodePublicKey: nodePublicKey,
	}
	if err = apPayload.SerializeUnsigned(buf, payload.ActivateProducerVersion); err != nil {
		return err
	}
	signature, err := acc.Sign(buf.Bytes())
	if err != nil {
		return err
	}
	apPayload.Signature = signature

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.ActivateProducer,
		0,
		apPayload,
		nil,
		nil,
		nil,
		0,
		[]*pg.Program{},
	)

	OutputTx(0, 0, txn)

	return nil
}

var registerproducer = cli.Command{
	Name:  "register",
	Usage: "Build a tx to register producer",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNodePublicKeyFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
		cmdcom.TransactionNetAddressFlag,
		cmdcom.TransactionStakeUntilFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createRegisterProducerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var updateproducer = cli.Command{
	Name:  "update",
	Usage: "Build a tx to update producer",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionFeeFlag,
		cmdcom.TransactionNodePublicKeyFlag,
		cmdcom.TransactionNickNameFlag,
		cmdcom.TransactionUrlFlag,
		cmdcom.TransactionLocationFlag,
		cmdcom.TransactionNetAddressFlag,
		cmdcom.TransactionStakeUntilFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createUpdateProducerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var unregisterproducer = cli.Command{
	Name:  "unregister",
	Usage: "Build a tx to unregister producer",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createUnregisterProducerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var returndeposit = cli.Command{
	Name:  "returndeposit",
	Usage: "Build a tx to return deposit coin of producer",
	Flags: []cli.Flag{
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
		cmdcom.TransactionAmountFlag,
		cmdcom.TransactionFeeFlag,
	},
	Action: func(c *cli.Context) error {
		if err := createReturnProducerDepositTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var activateproducer = cli.Command{
	Name:  "activate",
	Usage: "Build a tx to activate producer which have been inactivated",
	Flags: []cli.Flag{
		cmdcom.TransactionNodePublicKeyFlag,
		cmdcom.AccountWalletFlag,
		cmdcom.AccountPasswordFlag,
	},
	Action: func(c *cli.Context) error {
		if err := CreateActivateProducerTransaction(c); err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		return nil
	},
}

var producer = cli.Command{
	Name:  "producer",
	Usage: "Create producer related transactions",
	Flags: []cli.Flag{},
	Subcommands: []cli.Command{
		registerproducer,
		updateproducer,
		unregisterproducer,
		returndeposit,
		activateproducer,
	},
	Action: func(c *cli.Context) error {
		cli.ShowSubcommandHelp(c)
		return nil
	},
}
