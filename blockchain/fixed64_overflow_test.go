// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package blockchain_test

import (
	"errors"
	"math"
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/transaction"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

const (
	exploitInputValue       common.Fixed64 = 2405641496
	exploitLargeOutputValue common.Fixed64 = 9223322038057596306
	exploitThirdOutputValue common.Fixed64 = 100000000000000
)

func TestExploitTransactionFeeIsRejected(t *testing.T) {
	tx := newTransferAssetTransaction(
		exploitLargeOutputValue,
		exploitLargeOutputValue,
		exploitThirdOutputValue,
	)
	references := newReferences(exploitInputValue)

	_, err := blockchain.GetTransactionFee(tx, references)
	if !errors.Is(err, common.ErrFixed64Overflow) {
		t.Fatalf("expected exploit output overflow, got %v", err)
	}
}

func TestCheckedTransactionFeeAcceptsNormalValues(t *testing.T) {
	tx := newTransferAssetTransaction(999500)

	fee, err := blockchain.GetTransactionFee(tx, newReferences(1000000))
	if err != nil {
		t.Fatalf("normal fee calculation failed: %v", err)
	}
	if fee != 500 {
		t.Fatalf("expected fee 500, got %d", fee)
	}
}

func TestCheckedTransactionFeeRejectsAlternativeOverflowShapes(t *testing.T) {
	testCases := []struct {
		name       string
		tx         interfaces.Transaction
		references map[*common2.Input]common2.Output
	}{
		{
			name: "multiple output wraps",
			tx: newTransferAssetTransaction(
				math.MaxInt64,
				math.MaxInt64,
				math.MaxInt64,
			),
			references: newReferences(1),
		},
		{
			name:       "input sum overflow",
			tx:         newTransferAssetTransaction(1),
			references: newReferences(math.MaxInt64, math.MaxInt64),
		},
		{
			name:       "negative output",
			tx:         newTransferAssetTransaction(-1),
			references: newReferences(1),
		},
		{
			name:       "negative input",
			tx:         newTransferAssetTransaction(1),
			references: newReferences(-1),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := blockchain.GetTransactionFee(testCase.tx,
				testCase.references); err == nil {
				t.Fatal("expected checked fee calculation to fail")
			}
		})
	}
}

func TestGetTxFeePropagatesOverflow(t *testing.T) {
	tx := newTransferAssetTransaction(math.MaxInt64, 1)

	_, err := blockchain.GetTxFee(tx, core.ELAAssetID, newReferences(1))
	if !errors.Is(err, common.ErrFixed64Overflow) {
		t.Fatalf("expected per-asset overflow, got %v", err)
	}
}

func newTransferAssetTransaction(values ...common.Fixed64) interfaces.Transaction {
	outputs := make([]*common2.Output, 0, len(values))
	for _, value := range values {
		outputs = append(outputs, &common2.Output{
			AssetID: core.ELAAssetID,
			Value:   value,
		})
	}

	return transaction.CreateTransaction(
		common2.TxVersionDefault,
		common2.TransferAsset,
		0,
		new(payload.TransferAsset),
		nil,
		nil,
		outputs,
		0,
		nil,
	)
}

func newReferences(values ...common.Fixed64) map[*common2.Input]common2.Output {
	references := make(map[*common2.Input]common2.Output, len(values))
	for i, value := range values {
		input := &common2.Input{
			Previous: common2.OutPoint{Index: uint16(i)},
		}
		references[input] = common2.Output{
			AssetID: core.ELAAssetID,
			Value:   value,
		}
	}
	return references
}
