package blockchain

import (
	"crypto/rand"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/config"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/log"
	"os"
	"testing"
)

var txPool TxPool

func TestTxPoolInit(t *testing.T) {
	log.Init(log.Path, os.Stdout)
	FoundationAddress = config.Parameters.Configuration.FoundationAddress

	if FoundationAddress == "" {
		FoundationAddress = "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	}

	chainStore, err := NewChainStore()
	if err != nil {
		t.Fatal("open LedgerStore err:", err)
		os.Exit(1)
	}
	defer chainStore.Close()

	err = Init(chainStore)
	if err != nil {
		t.Fatal(err, "BlockChain generate failed")
	}

	txPool.Init()
}

func TestTxPool_VerifyDuplicateSidechainTx(t *testing.T) {
	// 1. Generate a withdraw transaction
	txn1 := new(core.Transaction)
	txn1.TxType = core.WithdrawAsset
	txn1.Payload = &core.PayloadWithdrawAsset{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHash: []string{
			"8a6cb4b5ff1a4f8368c6513a536c663381e3fdeff738e9b437bd8fce3fb30b62",
			"cc62e14f5f9526b7f4ff9d34dcd0643dacb7886707c57f49ec97b95ec5c4edac",
		},
	}

	// 2. Add sidechain Tx to pool
	witPayload := txn1.Payload.(*core.PayloadWithdrawAsset)
	for _, hash := range witPayload.SideChainTransactionHash {
		success := txPool.addSidechainTx(hash)
		if !success {
			t.Error("Add sidechain Tx to pool failed")
		}
	}

	// 3. Generate a withdraw transaction with duplicate sidechain Tx which already in the pool
	txn2 := new(core.Transaction)
	txn2.TxType = core.WithdrawAsset
	txn2.Payload = &core.PayloadWithdrawAsset{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHash: []string{
			"8a6cb4b5ff1a4f8368c6513a536c663381e3fdeff738e9b437bd8fce3fb30b62", // duplicate sidechain Tx
		},
	}

	// 4. Run verifyDuplicateSidechainTx
	err := txPool.verifyDuplicateSidechainTx(txn2)
	if err == nil {
		t.Error("Should find the duplicate sidechain tx")
	}
}

func TestTxPool_CleanSidechainTx(t *testing.T) {
	// 1. Generate some withdraw transactions
	txn1 := new(core.Transaction)
	txn1.TxType = core.WithdrawAsset
	txn1.Payload = &core.PayloadWithdrawAsset{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHash: []string{
			"300db7783393a6f60533c1223108445df57de4fb4842f84f55d07df57caa0c7d",
			"d6c2cb8345a8fe4af0d103cc4e40dbb0654bb169a85bb8cc57923d0c72f3658f",
		},
	}

	txn2 := new(core.Transaction)
	txn2.TxType = core.WithdrawAsset
	txn2.Payload = &core.PayloadWithdrawAsset{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHash: []string{
			"326218253e6feaa21e3521eff27418b942a5fbd45347505f3e5aca0463baffe2",
		},
	}

	txn3 := new(core.Transaction)
	txn3.TxType = core.WithdrawAsset
	txn3.Payload = &core.PayloadWithdrawAsset{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHash: []string{
			"645b614eaaa0a1bfd7015d88f3c1343048343924fc105e403b735ba754caa8db",
			"9dcad6d4ec2851bf522ddd301c7567caf98554a82a0bcce866de80b503909642",
		},
	}
	txns := []*core.Transaction{txn1, txn2, txn3}

	// 2. Add to sidechain txs pool
	for _, txn := range txns {
		witPayload := txn.Payload.(*core.PayloadWithdrawAsset)
		for _, hash := range witPayload.SideChainTransactionHash {
			success := txPool.addSidechainTx(hash)
			if !success {
				t.Error("Add to sidechain tx pool failed")
			}
		}
	}

	// Verify sidechain tx pool state
	for _, txn := range txns {
		err := txPool.verifyDuplicateSidechainTx(txn)
		if err == nil {
			t.Error("Should find the duplicate sidechain tx")
		}
	}

	// 3. Run cleanSidechainTx
	txPool.cleanSidechainTx(txns)

	// Verify sidechian tx pool state
	for _, txn := range txns {
		err := txPool.verifyDuplicateSidechainTx(txn)
		if err != nil {
			t.Error("Should not find the duplicate sidechain tx")
		}
	}
}

func TestTxPool_ReplaceDuplicateSideminingTx(t *testing.T) {
	var sideBlockHash1 common.Uint256
	var sideBlockHash2 common.Uint256
	var sideGenesisHash common.Uint256
	rand.Read(sideBlockHash1[:])
	rand.Read(sideBlockHash2[:])
	rand.Read(sideGenesisHash[:])

	txn1 := new(core.Transaction)
	txn1.TxType = core.SideMining
	txn1.Payload = &core.PayloadSideMining{
		SideBlockHash:   sideBlockHash1,
		SideGenesisHash: sideGenesisHash,
		BlockHeight:     100,
	}

	ok := txPool.addToTxList(txn1)
	if !ok {
		t.Error("Add sidemining txn1 to txpool failed")
	}

	txn2 := new(core.Transaction)
	txn2.TxType = core.SideMining
	txn2.Payload = &core.PayloadSideMining{
		SideBlockHash:   sideBlockHash2,
		SideGenesisHash: sideGenesisHash,
		BlockHeight:     100,
	}
	txPool.replaceDuplicateSideminingTx(txn2)
	ok = txPool.addToTxList(txn2)
	if !ok {
		t.Error("Add sidemining txn2 to txpool failed")
	}

	if txn := txPool.GetTransaction(txn1.Hash()); txn != nil {
		t.Errorf("Txn1 should be replaced")
	}

	if txn := txPool.GetTransaction(txn2.Hash()); txn == nil {
		t.Errorf("Txn2 should be added in txpool")
	}
}
