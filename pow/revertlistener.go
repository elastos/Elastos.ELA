// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package pow

import (
	"time"
)

const CheckRevertToPOWInterval = time.Minute

func (pow *Service) ListenForRevert() {
	go func() {
		for {
			time.Sleep(CheckRevertToPOWInterval)
			if pow.chain.BestChain.Height < pow.chainParams.RevertToPOWStartHeight {
				continue
			}
			if pow.arbiters.IsInPOWMode() {
				continue
			}
			lastBlockTimestamp := int64(pow.arbiters.GetLastBlockTimestamp())
			localTimestamp := pow.chain.TimeSource.AdjustedTime().Unix()
			if localTimestamp-lastBlockTimestamp < pow.chainParams.RevertToPOWNoBlockTime {
				continue
			}


			// todo refactor me
			//revertToPOWPayload := payload.RevertToPOW{
			//	Type:          payload.NoBlock,
			//	WorkingHeight: pow.chain.BestChain.Height + 1,
			//}
			//tx := &transactions.BaseTransaction{
			//	Version:        common.TxVersion09,
			//	TxType:         common.RevertToPOW,
			//	PayloadVersion: payload.RevertToPOWVersion,
			//	Payload:        &revertToPOWPayload,
			//	Attributes:     []*common.Attribute{},
			//	Programs:       []*program.Program{},
			//	LockTime:       0,
			//}
			//err := pow.txMemPool.AppendToTxPoolWithoutEvent(tx)
			//if err != nil {
			//	log.Error("failed to append revertToPOW transaction to " +
			//		"transaction pool, err:" + err.Error())
			//}
		}
	}()
}
