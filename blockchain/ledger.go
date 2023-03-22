// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package blockchain

import (
	"errors"
	"strconv"

	. "github.com/elastos/Elastos.ELA/common"
	. "github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

var (
	FoundationAddress Uint168
	DefaultLedger     *Ledger
)

// Ledger - the struct for ledger
type Ledger struct {
	Blockchain  *BlockChain
	Store       IChainStore
	Arbitrators state.Arbitrators
	Committee   *crstate.Committee
}

func (l *Ledger) GetAmount(programHash Uint168) (Fixed64, error) {
	amount := Fixed64(0)
	utxos, err := l.Store.GetFFLDB().GetUTXO(&programHash)
	if err != nil {
		return amount, err
	}
	for _, utxo := range utxos {
		amount += utxo.Value
	}
	return amount, nil
}

// check weather the transaction contains the doubleSpend.
func (l *Ledger) IsDoubleSpend(Tx interfaces.Transaction) bool {
	return DefaultLedger.Store.IsDoubleSpend(Tx)
}

// Get Block With Height.
func (l *Ledger) GetBlockWithHeight(height uint32) (*Block, error) {
	temp, err := l.Blockchain.GetBlockHash(height)
	if err != nil {
		return nil, errors.New("[Ledger],GetBlockWithHeight failed with height=" + strconv.Itoa(int(height)))
	}
	bk, err := DefaultLedger.Store.GetFFLDB().GetBlock(temp)
	if err != nil {
		return nil, errors.New("[Ledger],GetBlockWithHeight failed with hash=" + temp.String())
	}
	return bk.Block, nil
}

// Get block with block hash.
func (l *Ledger) GetBlockWithHash(hash Uint256) (*Block, error) {
	bk, err := l.Store.GetFFLDB().GetBlock(hash)
	if err != nil {
		return nil, errors.New("[Ledger],GetBlockWithHeight failed with hash=" + hash.String())
	}
	return bk.Block, nil
}

// Get transaction with hash.
func (l *Ledger) GetTransactionWithHash(hash Uint256) (interfaces.Transaction, error) {
	tx, _, err := l.Store.GetTransaction(hash)
	if err != nil {
		return nil, errors.New("[Ledger],GetTransactionWithHash failed with hash=" + hash.String())
	}
	return tx, nil
}

// Get local block chain height.
func (l *Ledger) GetLocalBlockChainHeight() uint32 {
	return l.Blockchain.GetHeight()
}

// Get blocks and confirms by given height range, if end equals zero will be treat as current highest block height
func (l *Ledger) GetDposBlocks(start, end uint32) ([]*DposBlock, error) {
	//todo complete me
	return nil, nil
}

// Append blocks and confirms directly
func (l *Ledger) AppendDposBlocks(confirms []*DposBlock) error {
	//todo complete me
	return nil
}
