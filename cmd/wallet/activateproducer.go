package wallet

import (
	"bytes"
	"fmt"
	"errors"
	"os"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	pg "github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/account"

	"github.com/urfave/cli"
)

var activateproducer = cli.Command {
	Name:  "activateproducer",
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
