// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

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
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/utils/http"

	"github.com/urfave/cli"
)

var txCommand = []cli.Command{
	{
		Category:    "Transaction",
		Name:        "buildtx",
		Usage:       "Build a transaction",
		Description: "use --to --amount --fee to create a transaction",
		Flags: []cli.Flag{
			cmdcom.TransactionFromFlag,
			cmdcom.TransactionToFlag,
			cmdcom.TransactionToManyFlag,
			cmdcom.TransactionAmountFlag,
			cmdcom.TransactionFeeFlag,
			cmdcom.TransactionOutputLockFlag,
			cmdcom.TransactionTxLockFlag,
			cmdcom.AccountWalletFlag,
		},
		Subcommands: buildTxCommand,
		Action:      buildTx,
	},
	{
		Category:    "Transaction",
		Name:        "signtx",
		Usage:       "Sign a transaction",
		Description: "use --file or --hex to specify the transaction file path or content",
		Flags: []cli.Flag{
			cmdcom.TransactionHexFlag,
			cmdcom.TransactionFileFlag,
			cmdcom.AccountWalletFlag,
			cmdcom.AccountPasswordFlag,
		},
		Action: signTx,
	},
	{
		Category: "Transaction",
		Name:     "signdigest",
		Usage:    "sign digest",
		Flags: []cli.Flag{
			cmdcom.AccountWalletFlag,
			cmdcom.AccountPasswordFlag,
			cmdcom.TransactionDigestFlag,
		},
		Action: signDigest,
	},
	{
		Category: "Transaction",
		Name:     "verifydigest",
		Usage:    "verify digest",
		Flags: []cli.Flag{
			cmdcom.AccountWalletFlag,
			cmdcom.AccountPasswordFlag,
			cmdcom.TransactionDigestFlag,
			cmdcom.TransactionSignatureFlag,
		},
		Action: verifyDigest,
	},
	{
		Category:    "Transaction",
		Name:        "sendtx",
		Usage:       "Send a transaction",
		Description: "use --file or --hex to specify the transaction file path or content",
		Flags: []cli.Flag{
			cmdcom.TransactionHexFlag,
			cmdcom.TransactionFileFlag,
		},
		Action: sendTx,
	},
	{
		Category: "Transaction",
		Name:     "showtx",
		Usage:    "Show info of raw transaction",
		Flags: []cli.Flag{
			cmdcom.TransactionHexFlag,
			cmdcom.TransactionFileFlag,
		},
		Action: showTx,
	},
}

var buildTxCommand = []cli.Command{
	dpossv2claimreward,
	vote,
	crosschain,
	producer,
	crc,
	proposal,
	stake,
	returnvotes,
	dposV2Vote,
	dposV2VoteRenew,
}

func getTransactionHex(c *cli.Context) (string, error) {
	if filePath := strings.TrimSpace(c.String("file")); filePath != "" {
		return cmdcom.ReadFile(filePath)
	}

	content := strings.TrimSpace(c.String("hex"))
	if content == "" {
		return "", errors.New("transaction hex string is empty")
	}

	return content, nil
}

func buildTx(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}
	if err := CreateTransaction(c); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	return nil
}

func signTx(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}
	walletPath := c.String("wallet")
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}

	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}

	txHex, err := getTransactionHex(c)
	if err != nil {
		return err
	}
	rawData, err := common.HexStringToBytes(txHex)
	if err != nil {
		return errors.New("decode transaction content failed")
	}

	r := bytes.NewReader(rawData)
	txn, err := functions.GetTransactionByBytes(r)
	if err != nil {
		return errors.New("invalid transaction")
	}
	err = txn.Deserialize(r)
	if err != nil {
		return errors.New("deserialize transaction failed")
	}

	if len(txn.Programs()) == 0 {
		return errors.New("no program found in transaction")
	}

	haveSign, needSign, err := crypto.GetSignStatus(txn.Programs()[0].Code, txn.Programs()[0].Parameter)
	if err != nil {
		return err
	}
	if haveSign == needSign {
		return errors.New("transaction was fully signed, no need more sign")
	}

	txnSigned, err := client.Sign(txn)
	if err != nil {
		return err
	}

	haveSign, needSign, _ = crypto.GetSignStatus(txn.Programs()[0].Code, txn.Programs()[0].Parameter)
	fmt.Println("[", haveSign, "/", needSign, "] BaseTransaction was successfully signed")

	OutputTx(haveSign, needSign, txnSigned)

	return nil
}

func signDigest(c *cli.Context) error {
	var name string

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}

	sDigest := c.String(cmdcom.TransactionDigestFlag.Name)
	digest, err := common.Uint256FromReversedHexString(sDigest)
	if err != nil {
		return errors.New("invalid digest: " + err.Error())
	}

	var acc *account.Account
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}

	acc = client.GetMainAccount()

	sign, err := acc.SignDigest(digest.Bytes())
	if err != nil {
		return errors.New("sign failed: " + err.Error())
	}

	fmt.Println("Signature: ", common.BytesToHexString(sign))
	return nil
}

func verifyDigest(c *cli.Context) error {
	var name string

	name = strings.Split(cmdcom.AccountWalletFlag.Name, ",")[0]
	walletPath := c.String(name)
	if walletPath == "" {
		return errors.New(fmt.Sprintf("use --%s to specify wallet path", name))
	}
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}

	sDigest := c.String(cmdcom.TransactionDigestFlag.Name)
	digest, err := common.Uint256FromReversedHexString(sDigest)
	if err != nil {
		return errors.New("invalid digest: " + err.Error())
	}

	sSign := c.String(cmdcom.TransactionSignatureFlag.Name)
	sign, err := common.HexStringToBytes(sSign)
	if err != nil {
		return errors.New("invalid signature: " + err.Error())
	}

	var acc *account.Account
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}

	acc = client.GetMainAccount()

	err = crypto.VerifyDigest(*acc.PublicKey, digest.Bytes(), sign)
	if err != nil {
		fmt.Println("verify digest: false")
	} else {
		fmt.Println("verify digest: true")
	}

	return nil
}

func sendTx(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	txHex, err := getTransactionHex(c)
	if err != nil {
		return err
	}

	result, err := cmdcom.RPCCall("sendrawtransaction", http.Params{"data": txHex})
	if err != nil {
		return err
	}
	fmt.Println(result.(string))

	return nil
}

func showTx(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	txHex, err := getTransactionHex(c)
	if err != nil {
		return err
	}

	txBytes, err := common.HexStringToBytes(txHex)
	if err != nil {
		return err
	}
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	if err := txn.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return err
	}

	fmt.Println(txn.String())

	return nil
}
