// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package wallet

import (
	"fmt"

	"github.com/elastos/Elastos.ELA/account"
	cmdcom "github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"

	"github.com/urfave/cli"
	"golang.org/x/crypto/sha3"
)

var toolsCommand = []cli.Command{
	{
		Category: "Tools",
		Name:     "signoracleid",
		Usage:    "Create an account",
		Flags: []cli.Flag{
			cmdcom.AccountWalletFlag,
			cmdcom.AccountPasswordFlag,
			cmdcom.OracleIdFlag,
			cmdcom.JobIdFlag,
		},
		Action: signOracleId,
	},
}

func signOracleId(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}
	walletPath := c.String("wallet")
	oracleIdHex := c.String("oracleid")
	jobId := c.String("jobid")
	password, err := cmdcom.GetFlagPassword(c)
	if err != nil {
		return err
	}
	client, err := account.Open(walletPath, password)
	if err != nil {
		return err
	}
	oracleId, err := common.HexStringToBytes(oracleIdHex)
	if err != nil {
		return err
	}

	data := make([]byte, len(oracleId)+32)
	copy(data[0:len(oracleId)], oracleId)
	copy(data[len(oracleId):], Keccak256([]byte(jobId)))
	signature, err := crypto.Sign(client.GetMainAccount().PrivKey(), data)
	if err != nil {
		return err
	}
	fmt.Println("Signature:", common.BytesToHexString(signature))

	return nil
}

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
