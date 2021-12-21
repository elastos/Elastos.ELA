package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/urfave/cli"
)

var registerproducer = cli.Command {
	Name:  "registerproducer",
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

func createRegisterProducerTransaction(c *cli.Context) error {
	var name string

	name = cmdcom.TransactionAmountFlag.Name
	amountStr := c.String(name)
	if amountStr == "" {
		return errors.New(fmt.Sprintf("use --%s to specify transfer amount", name))
	}
	amount, err := common.StringToFixed64(amountStr)
	if err != nil {
		return errors.New("invalid transaction amount")
	}

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
	programHash, err := contract.PublicKeyToDepositProgramHash(ownerPublicKey)
	if (err != nil) {
		return err
	}
	to, err := programHash.ToAddress()
	if (err != nil) {
		return err
	}
	outputs = []*OutputInfo{{to, amount}}

	var nodePublicKey []byte
	name = cmdcom.TransactionNodePublicKeyFlag.Name
	nodePublicKeyStr := c.String(name)
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
	txn, err = createTransaction(walletPath, "", *fee, 0, 0, common2.RegisterProducer,
		payload.ProducerInfoDposV2Version, p, outputs...)
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	OutputTx(0, 1, txn)

	return nil
}