package transaction

import (
	"testing"

	uti_tx "github.com/elastos/Elastos.ELA.Utility/core/transaction"
	uti_payload "github.com/elastos/Elastos.ELA.Utility/core/transaction/payload"
	p "github.com/elastos/Elastos.ELA/core/transaction/payload"
)

func TestPayloadFactoryNodeImpl_Name(t *testing.T) {
	name := uti_tx.PayloadFactorySingleton.Name(uti_tx.CoinBase)
	if name != "CoinBase" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "CoinBase", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(uti_tx.RegisterAsset)
	if name != "RegisterAsset" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "RegisterAsset", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(uti_tx.TransferAsset)
	if name != "TransferAsset" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "TransferAsset", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(uti_tx.Record)
	if name != "Record" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Record", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(uti_tx.Deploy)
	if name != "Deploy" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Deploy", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(SideMining)
	if name != "SideMining" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Deploy", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(IssueToken)
	if name != "IssueToken" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Deploy", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(WithdrawToken)
	if name != "WithdrawToken" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Deploy", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(TransferCrossChainAsset)
	if name != "TransferCrossChainAsset" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Deploy", name)
	}

	name = uti_tx.PayloadFactorySingleton.Name(0x09)
	if name != "Unknown" {
		t.Errorf("TransactionTypeName: [%v], actually: [%v]", "Unknown", name)
	}
}

func TestPayloadFactoryNodeImpl_Create(t *testing.T) {
	payload, err := uti_tx.PayloadFactorySingleton.Create(uti_tx.CoinBase)
	if _, ok := payload.(*uti_payload.CoinBase); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(uti_tx.RegisterAsset)
	if _, ok := payload.(*uti_payload.RegisterAsset); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(uti_tx.TransferAsset)
	if _, ok := payload.(*uti_payload.TransferAsset); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(uti_tx.Record)
	if _, ok := payload.(*uti_payload.Record); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(uti_tx.Deploy)
	if _, ok := payload.(*uti_payload.DeployCode); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(SideMining)
	if _, ok := payload.(*p.SideMining); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(WithdrawToken)
	if _, ok := payload.(*p.WithdrawToken); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(TransferCrossChainAsset)
	if _, ok := payload.(*p.TransferCrossChainAsset); !ok {
		t.Error("Payload create error.")
	}
	if err != nil {
		t.Error("Unexpect error.")
	}

	payload, err = uti_tx.PayloadFactorySingleton.Create(0x09)
	if payload != nil {
		t.Error("Payload create error.")
	}
	if err == nil {
		t.Error("Expect an error.")
	}
}
