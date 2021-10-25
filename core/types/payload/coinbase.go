// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"fmt"
	"io"

	elaerr "github.com/elastos/Elastos.ELA/errors"

	"github.com/elastos/Elastos.ELA/common"
)

const (
	// MaxPayloadDataSize is the maximum allowed length of payload data.
	MaxPayloadDataSize = 1024 * 1024 // 1MB
)

const CoinBaseVersion byte = 0x04

type CoinBase struct {
	DefaultChecker

	Content []byte
}

func (a *CoinBase) Data(version byte) []byte {
	return a.Content
}

func (a *CoinBase) Serialize(w io.Writer, version byte) error {
	return common.WriteVarBytes(w, a.Content)
}

func (a *CoinBase) Deserialize(r io.Reader, version byte) error {
	temp, err := common.ReadVarBytes(r, MaxPayloadDataSize,
		"payload coinbase data")
	a.Content = temp
	return err
}

// todo add description
func (a *CoinBase) SpecialCheck(p *CheckParameters) (elaerr.ELAError, bool) {
	// todo special check, all check witch used isCoinbase function, need to move here.
	//if p.BlockHeight >= p.CRCommitteeStartHeight {
	//	if p.ConsensusAlgorithm == 0x01 {
	//		if !txn.Outputs[0].ProgramHash.IsEqual(p.DestroyELAAddress) {
	//			return elaerr.Simple(elaerr.ErrTxInvalidOutput,
	//				errors.New("first output address should be "+
	//					"DestroyAddress in POW consensus algorithm")), true
	//		}
	//	} else {
	//		if !txn.Outputs[0].ProgramHash.IsEqual(p.CRAssetsAddress) {
	//			return elaerr.Simple(elaerr.ErrTxInvalidOutput,
	//				errors.New("first output address should be CR assets address")), true
	//		}
	//	}
	//} else if !txn.Outputs[0].ProgramHash.IsEqual(p.FoundationAddress) {
	//	return elaerr.Simple(elaerr.ErrTxInvalidOutput,
	//		errors.New("first output address should be foundation address")), true
	//}
	//

	fmt.Println("CoinBase self check")
	return nil, true
}

func (a *CoinBase) ContextCheck(para *CheckParameters) elaerr.ELAError {

	if err := a.CheckTxHeightVersion(para); err != nil {
		return elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	//// check if duplicated with transaction in ledger
	//if exist := b.db.IsTxHashDuplicate(txn.Hash()); exist {
	//	log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
	//	return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	//}
	if exist := a.IsTxHashDuplicate(para.TxHash); exist {
		//log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	firstErr, end := a.SpecialCheck(para)
	if end {
		return firstErr
	}

	return nil
}
