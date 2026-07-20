// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package transaction

import (
	"errors"
	"math"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

func TestActivateProducerRejectsOutputOverflow(t *testing.T) {
	chainParams := config.GetDefaultParams()
	tx := CreateTransaction(
		common2.TxVersionDefault,
		common2.ActivateProducer,
		0,
		new(payload.ActivateProducer),
		nil,
		nil,
		[]*common2.Output{
			{AssetID: core.ELAAssetID, Value: 9223322038057596306},
			{AssetID: core.ELAAssetID, Value: 9223322038057596306},
			{AssetID: core.ELAAssetID, Value: 100000000000000},
		},
		0,
		nil,
	)

	err := tx.SanityCheck(&TransactionParameters{
		Transaction: tx,
		BlockHeight: chainParams.DPoSConfiguration.NFTStartHeight + 1,
		Config:      chainParams,
	})
	if err == nil {
		t.Fatal("expected ActivateProducer output overflow to be rejected")
	}
	if err.Code() != elaerr.ErrTxInvalidOutput {
		t.Fatalf("expected invalid-output error, got %d", err.Code())
	}
	if !errors.Is(err.InnerError(), common.ErrFixed64Overflow) {
		t.Fatalf("expected Fixed64 overflow, got %v", err.InnerError())
	}
}

func TestFeePerKBOverflowDoesNotMutateTransaction(t *testing.T) {
	chainParams := config.GetDefaultParams()
	tx := CreateTransaction(
		common2.TxVersionDefault,
		common2.TransferAsset,
		0,
		new(payload.TransferAsset),
		nil,
		nil,
		[]*common2.Output{{AssetID: core.ELAAssetID, Value: 1}},
		0,
		nil,
	)
	tx.SetParameters(&TransactionParameters{
		Transaction: tx,
		BlockHeight: 0,
		Config:      chainParams,
	})
	references := map[*common2.Input]common2.Output{
		new(common2.Input): {
			AssetID: core.ELAAssetID,
			Value:   math.MaxInt64,
		},
	}

	err := tx.CheckTransactionFee(references)
	if !errors.Is(err, common.ErrFixed64Overflow) {
		t.Fatalf("expected fee-rate overflow, got %v", err)
	}
	if tx.Fee() != 0 || tx.FeePerKB() != 0 {
		t.Fatalf("failed calculation mutated fee fields: fee=%d feePerKB=%d",
			tx.Fee(), tx.FeePerKB())
	}
}
