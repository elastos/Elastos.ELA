// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
)

func TestCheckFrozenAddresses(t *testing.T) {
	const activationHeight uint32 = 100

	programHash, err := common.Uint168FromAddress(config.ExploitIntermediateFrozenAddress)
	if err != nil {
		t.Fatalf("decode frozen address: %v", err)
	}
	frozenAddresses := []config.FrozenAddress{{
		Address:            config.ExploitIntermediateFrozenAddress,
		DisableStartHeight: activationHeight,
		ProgramHash:        programHash,
	}}

	txn, err := GetTransaction(common2.TransferAsset)
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	txn.SetOutputs([]*common2.Output{{
		ProgramHash: *programHash,
	}})

	references := map[*common2.Input]common2.Output{
		&common2.Input{}: {ProgramHash: *programHash},
	}

	if err := checkFrozenAddresses(txn, references, activationHeight-1, frozenAddresses); err != nil {
		t.Fatalf("expected frozen address to be allowed before activation, got %v", err)
	}

	err = checkFrozenAddresses(txn, references, activationHeight, frozenAddresses)
	if err == nil || err.Error() != "cannot use utxo from the frozen address "+config.ExploitIntermediateFrozenAddress {
		t.Fatalf("unexpected input error: %v", err)
	}

	normalHash, err := common.Uint168FromAddress("EJMzC16Eorq9CuFCGtyMrq4Jmgw9jYCHQR")
	if err != nil {
		t.Fatalf("decode normal address: %v", err)
	}
	references = map[*common2.Input]common2.Output{
		&common2.Input{}: {ProgramHash: *normalHash},
	}
	err = checkFrozenAddresses(txn, references, activationHeight, frozenAddresses)
	if err == nil || err.Error() != "cannot send to the frozen address "+config.ExploitIntermediateFrozenAddress {
		t.Fatalf("unexpected output error: %v", err)
	}

	txn.SetOutputs([]*common2.Output{{ProgramHash: *normalHash}})
	if err := checkFrozenAddresses(txn, references, activationHeight, frozenAddresses); err != nil {
		t.Fatalf("expected normal transfer to pass, got %v", err)
	}

	if err := checkFrozenAddresses(txn, references, activationHeight, nil); err != nil {
		t.Fatalf("expected empty frozen list to pass, got %v", err)
	}
}
