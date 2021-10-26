// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"errors"
	"fmt"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
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
func (a *CoinBase) SpecialCheck(para *CheckParameters) (elaerr.ELAError, bool) {
	// todo special check, all check witch used isCoinbase function, need to move here.
	if para.BlockHeight >= para.CRCommitteeStartHeight {
		if para.ConsensusAlgorithm == 0x01 {
			if !para.Outputs[0].ProgramHash.IsEqual(para.DestroyELAAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be "+
						"DestroyAddress in POW consensus algorithm")), true
			}
		} else {
			if !para.Outputs[0].ProgramHash.IsEqual(para.CRAssetsAddress) {
				return elaerr.Simple(elaerr.ErrTxInvalidOutput,
					errors.New("first output address should be CR assets address")), true
			}
		}
	} else if !para.Outputs[0].ProgramHash.IsEqual(para.FoundationAddress) {
		return elaerr.Simple(elaerr.ErrTxInvalidOutput,
			errors.New("first output address should be foundation address")), true
	}

	fmt.Println("CoinBase self check")
	return nil, true
}

func (a *CoinBase) ContextCheck(para *CheckParameters) (map[*common2.Input]common2.Output, elaerr.ELAError) {

	if err := a.CheckTxHeightVersion(para); err != nil {
		return nil, elaerr.Simple(elaerr.ErrTxHeightVersion, nil)
	}

	//// check if duplicated with transaction in ledger
	//if exist := b.db.IsTxHashDuplicate(txn.Hash()); exist {
	//	log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
	//	return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	//}
	if exist := a.IsTxHashDuplicate(para.TxHash); exist {
		//log.Warn("[CheckTransactionContext] duplicate transaction check failed.")
		return nil, elaerr.Simple(elaerr.ErrTxDuplicate, nil)
	}

	firstErr, end := a.SpecialCheck(para)
	if end {
		return nil, firstErr
	}

	return nil, nil
}
