// Copyright (c) 2017-2019 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//
package process

import (
	"fmt"
	"path"
	"testing"

	genchain "github.com/elastos/Elastos.ELA/benchmark/generator/chain"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/utils/test"
)

var (
	singleBlockGen    = newBlockChain()
	singleBlockParams ProcessParams
)

type ProcessParams struct {
	Ledger            blockchain.Ledger
	FoundationAddress common.Uint168
}

func Benchmark_SingleBlock_ProcessBlock(b *testing.B) {
	LoadParams(&singleBlockParams)

	currentHeight := singleBlockGen.GetChain().GetHeight()
	err := singleBlockGen.Generate(currentHeight + 1)
	if err != nil {
		b.Error(err)
	}
}

func newBlockChain() *genchain.DataGen {
	gen, err := genchain.LoadDataGen(
		path.Join(test.DataDir, genchain.TxRepoFile))
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	SaveParams(&singleBlockParams)
	return gen
}

func SaveParams(params *ProcessParams) {
	params.Ledger = *blockchain.DefaultLedger
	params.FoundationAddress = blockchain.FoundationAddress
}

func LoadParams(params *ProcessParams) {
	blockchain.DefaultLedger = &params.Ledger
	blockchain.FoundationAddress = params.FoundationAddress
}