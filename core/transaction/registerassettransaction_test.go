// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"strconv"
)

func (s *transactionSuite) TestRegisterAssetTransaction_SerializeDeserialize() {
	txn := randomOldVersionTransaction(true, byte(common2.RegisterAsset), s.InputNum, s.OutputNum, s.AttrNum, s.ProgramNum)
	txn.SetPayload(&payload.RegisterAsset{
		Asset: payload.Asset{
			Name:        "test name",
			Description: "test desc",
			Precision:   byte(rand.Uint32()),
			AssetType:   payload.AssetType(rand.Uint32()),
			RecordType:  payload.AssetRecordType(rand.Uint32()),
		},
		Amount:     common.Fixed64(rand.Int63()),
		Controller: *randomUint168(),
	})

	serializedData := new(bytes.Buffer)
	txn.Serialize(serializedData)

	txn2, err := functions.GetTransactionByBytes(serializedData)
	if err != nil {
		s.Assert()
	}
	txn2.Deserialize(serializedData)

	assertOldVersionTxEqual(true, &s.Suite, txn, txn2, s.InputNum, s.OutputNum, s.AttrNum, s.ProgramNum)

	p1 := txn.Payload().(*payload.RegisterAsset)
	p2 := txn2.Payload().(*payload.RegisterAsset)

	s.Equal(p1.Asset.Name, p2.Asset.Name)
	s.Equal(p1.Asset.Description, p2.Asset.Description)
	s.Equal(p1.Asset.Precision, p2.Asset.Precision)
	s.Equal(p1.Asset.AssetType, p2.Asset.AssetType)
	s.Equal(p1.Asset.RecordType, p2.Asset.RecordType)
	s.Equal(p1.Amount, p2.Amount)
	s.True(p1.Controller.IsEqual(p2.Controller))
}

func (s *transactionSuite) TestTransferAssert_SerializeDeserialize() {
	txn := randomOldVersionTransaction(true, byte(common2.TransferAsset), s.InputNum, s.OutputNum, s.AttrNum, s.ProgramNum)
	txn.SetPayload(&payload.TransferAsset{})

	serializedData := new(bytes.Buffer)
	txn.Serialize(serializedData)
	txn2, err := functions.GetTransactionByBytes(serializedData)
	if err != nil {
		s.Assert()
	}
	txn2.Deserialize(serializedData)

	assertOldVersionTxEqual(true, &s.Suite, txn, txn2, s.InputNum, s.OutputNum, s.AttrNum, s.ProgramNum)
}
func randomOldVersionTransaction(oldVersion bool, txType byte, inputNum, outputNum, attrNum, programNum int) interfaces.Transaction {
	txn := functions.CreateTransaction(
		common2.TransactionVersion(txType),
		common2.TxType(txType),
		byte(0),
		nil,
		make([]*common2.Attribute, 0),
		make([]*common2.Input, 0),
		make([]*common2.Output, 0),
		rand.Uint32(),
		make([]*program.Program, 0),
	)
	if !oldVersion {
		txn.SetVersion(common2.TxVersion09)
	}

	for i := 0; i < inputNum; i++ {
		txn.SetInputs(append(txn.Inputs(), &common2.Input{
			Sequence: rand.Uint32(),
			Previous: common2.OutPoint{
				TxID:  *randomUint256(),
				Index: uint16(rand.Uint32()),
			},
		}))
	}

	for i := 0; i < outputNum; i++ {
		output := &common2.Output{
			AssetID:     *randomUint256(),
			Value:       common.Fixed64(rand.Int63()),
			OutputLock:  rand.Uint32(),
			ProgramHash: *randomUint168(),
			Type:        0,
			Payload:     nil,
		}
		if !oldVersion {
			output.Type = common2.OTNone
			output.Payload = &outputpayload.DefaultOutput{}
		}
		txn.SetOutputs(append(txn.Outputs(), output))
	}

	validAttrUsage := []common2.AttributeUsage{common2.Nonce,
		common2.Script, common2.Memo, common2.Description,
		common2.DescriptionUrl, common2.Confirmations}
	for i := 0; i < attrNum; i++ {
		txn.SetAttributes(append(txn.Attributes(), &common2.Attribute{
			Usage: validAttrUsage[rand.Intn(len(validAttrUsage))],
			Data:  []byte(strconv.FormatUint(rand.Uint64(), 10)),
		}))
	}

	for i := 0; i < programNum; i++ {
		txn.SetPrograms(append(txn.Programs(), &program.Program{
			Code:      []byte(strconv.FormatUint(rand.Uint64(), 10)),
			Parameter: []byte(strconv.FormatUint(rand.Uint64(), 10)),
		}))
	}

	return txn
}
func assertOldVersionTxEqual(oldVersion bool, suite *suite.Suite, first, second interfaces.Transaction, inputNum, outputNum, attrNum, programNum int) {
	if oldVersion {
		suite.Equal(common2.TxVersionDefault, second.Version())
	} else {
		suite.Equal(first.Version(), second.Version())
	}
	suite.Equal(first.TxType(), second.TxType())
	suite.Equal(first.PayloadVersion(), second.PayloadVersion())
	suite.Equal(first.LockTime(), second.LockTime())

	suite.Equal(inputNum, len(first.Inputs()))
	suite.Equal(inputNum, len(second.Inputs()))
	for i := 0; i < inputNum; i++ {
		suite.Equal(first.Inputs()[i].Sequence, second.Inputs()[i].Sequence)
		suite.Equal(first.Inputs()[i].Previous.Index, second.Inputs()[i].Previous.Index)
		suite.True(first.Inputs()[i].Previous.TxID.IsEqual(second.Inputs()[i].Previous.TxID))
	}

	suite.Equal(outputNum, len(first.Outputs()))
	suite.Equal(outputNum, len(second.Outputs()))
	for i := 0; i < outputNum; i++ {
		suite.True(first.Outputs()[i].AssetID.IsEqual(second.Outputs()[i].AssetID))
		suite.Equal(first.Outputs()[i].Value, second.Outputs()[i].Value)
		suite.Equal(first.Outputs()[i].OutputLock, second.Outputs()[i].OutputLock)
		suite.True(first.Outputs()[i].ProgramHash.IsEqual(second.Outputs()[i].ProgramHash))

		if !oldVersion {
			suite.Equal(first.Outputs()[i].Type, second.Outputs()[i].Type)
		}
	}

	suite.Equal(attrNum, len(first.Attributes()))
	suite.Equal(attrNum, len(second.Attributes()))
	for i := 0; i < attrNum; i++ {
		suite.Equal(first.Attributes()[i].Usage, second.Attributes()[i].Usage)
		suite.True(bytes.Equal(first.Attributes()[i].Data, second.Attributes()[i].Data))
	}

	suite.Equal(programNum, len(first.Programs()))
	suite.Equal(programNum, len(second.Programs()))
	for i := 0; i < programNum; i++ {
		suite.True(bytes.Equal(first.Programs()[i].Code, second.Programs()[i].Code))
		suite.True(bytes.Equal(first.Programs()[i].Parameter, second.Programs()[i].Parameter))
	}
}
