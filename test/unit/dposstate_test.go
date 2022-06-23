// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package unit

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	state2 "github.com/elastos/Elastos.ELA/dpos/state"

	"github.com/stretchr/testify/assert"
)

func init() {
	testing.Init()

	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters
	config.DefaultParams = config.GetDefaultParams()
}

// mockBlock creates a block instance by the given height and transactions.
func mockBlock(height uint32, txs ...interfaces.Transaction) *types.Block {
	return &types.Block{
		Header: common2.Header{
			Height: height,
		},
		Transactions: txs,
	}
}

// mockRegisterProducerTx creates a register producer transaction with the given
// ProducerInfo.
func mockRegisterProducerTx(info *payload.ProducerInfo) interfaces.Transaction {
	tx := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		info,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return tx
}

// mockUpdateProducerTx creates a update producer transaction with the given
// ProducerInfo.
func mockUpdateProducerTx(info *payload.ProducerInfo) interfaces.Transaction {
	tx := functions.CreateTransaction(
		0,
		common2.UpdateProducer,
		0,
		info,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return tx
}

// mockCancelProducerTx creates a cancel producer transaction by the producer
// public key.
func mockCancelProducerTx(publicKey []byte) interfaces.Transaction {
	tx := functions.CreateTransaction(
		0,
		common2.CancelProducer,
		0,
		&payload.ProcessProducer{
			OwnerPublicKey: publicKey,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return tx
}

func mockActivateProducerTx(publicKey []byte) interfaces.Transaction {
	tx := functions.CreateTransaction(
		0,
		common2.ActivateProducer,
		0,
		&payload.ActivateProducer{
			NodePublicKey: publicKey,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return tx
}

// mockVoteTx creates a vote transaction with the producers public keys.
func mockVoteTx(publicKeys [][]byte) interfaces.Transaction {
	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(publicKeys))
	for _, pk := range publicKeys {
		candidateVotes = append(candidateVotes,
			outputpayload.CandidateVotes{pk, 0})
	}
	output := &common2.Output{
		Value: 100,
		Type:  common2.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				{outputpayload.Delegate, candidateVotes},
			},
		},
	}

	tx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{output},
		0,
		[]*program.Program{},
	)

	return tx
}

func mockNewDPoSVoteTx(publicKeys [][]byte) interfaces.Transaction {
	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(publicKeys))
	for i, pk := range publicKeys {
		candidateVotes = append(candidateVotes,
			outputpayload.CandidateVotes{
				Candidate: pk,
				Votes:     common.Fixed64((i + 1) * 10)})
	}
	output := &common2.Output{
		Value: 100,
		Type:  common2.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: outputpayload.VoteProducerAndCRVersion,
			Contents: []outputpayload.VoteContent{
				{outputpayload.Delegate, candidateVotes},
			},
		},
	}

	tx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{output},
		0,
		[]*program.Program{},
	)

	return tx
}

// mockVoteTx creates a cancel vote transaction with the previous vote
// transaction.
func mockCancelVoteTx(tx interfaces.Transaction) interfaces.Transaction {
	inputs := make([]*common2.Input, len(tx.Outputs()))
	for i := range tx.Outputs() {
		inputs[i] = &common2.Input{
			Previous: *common2.NewOutPoint(tx.Hash(), uint16(i)),
		}
	}

	tx = functions.CreateTransaction(
		common2.TxVersion09,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		inputs,
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	return tx
}

// mockIllegalBlockTx creates a illegal block transaction with the producer
// public key.
func mockIllegalBlockTx(publicKey []byte) interfaces.Transaction {
	tx := functions.CreateTransaction(
		common2.TxVersion09,
		common2.IllegalBlockEvidence,
		0,
		&payload.DPOSIllegalBlocks{
			Evidence: payload.BlockEvidence{
				Signers: [][]byte{publicKey},
			},
			CompareEvidence: payload.BlockEvidence{
				Signers: [][]byte{publicKey},
			},
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	return tx
}

// mockIllegalBlockTx creates a inactive Arbiters transaction with the
// producer public key.
func mockInactiveArbitratorsTx(publicKey []byte) interfaces.Transaction {
	tx := functions.CreateTransaction(
		0,
		common2.InactiveArbitrators,
		0,
		&payload.InactiveArbitrators{
			Arbitrators: [][]byte{publicKey},
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	return tx
}

func TestState_ProcessTransaction(t *testing.T) {
	params := config.GetDefaultParams()
	state := state2.NewState(&params, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// Test update producer.
	producers[0].NickName = "Updated"
	nodePublicKey := make([]byte, 33)
	rand.Read(nodePublicKey)
	producers[0].NodePublicKey = nodePublicKey
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil, false, 0)
	p := state.GetProducer(producers[0].NodePublicKey)
	if !assert.NotNil(t, p) {
		t.FailNow()
	}
	if !assert.Equal(t, "Updated", p.Info().NickName) {
		t.FailNow()
	}

	// Test cancel producer.
	tx = mockCancelProducerTx(producers[0].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil, false, 0)
	// at this point, we have 1 canceled, 3 pending, 6 active and 9 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 3, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}

	// Test vote producer.
	publicKeys := make([][]byte, 5)
	for i, p := range producers[0:5] {
		publicKeys[i] = p.OwnerPublicKey
	}
	tx = mockVoteTx(publicKeys)

	// Test new version vote producer.
	publicKeys2 := make([][]byte, 5)
	for i, p := range producers[5:10] {
		publicKeys2[i] = p.OwnerPublicKey
	}
	tx2 := mockNewDPoSVoteTx(publicKeys2)

	state.ProcessBlock(mockBlock(13, tx, tx2), nil, false, 0)

	for _, pk := range publicKeys {
		p := state.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(100), p.Votes()) {
			t.FailNow()
		}
	}

	for i, pk := range publicKeys2 {
		p := state.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64((i+1)*10), p.Votes()) {
			t.FailNow()
		}
	}

	// Test illegal producer.
	tx = mockIllegalBlockTx(producers[1].NodePublicKey)
	state.ProcessBlock(mockBlock(14, tx), nil, false, 0)
	// at this point, we have 1 canceled, 1 pending, 7 active, 1 illegal and 8 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 7, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_ProcessBlock(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 100 producers info.
	producers := make([]*payload.ProducerInfo, 100)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register 10 producers on one height.
	for i := 0; i < 10; i++ {
		txs := make([]interfaces.Transaction, 10)
		for i, p := range producers[i*10 : (i+1)*10] {
			txs[i] = mockRegisterProducerTx(p)
		}
		state.ProcessBlock(mockBlock(uint32(i+1), txs...), nil, false, 0)
	}
	// at this point, we have 50 pending, 50 active and 100 in total producers.
	if !assert.Equal(t, 50, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 50, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 100, len(state.GetProducers())) {
		t.FailNow()
	}

	// Update 10 producers.
	txs := make([]interfaces.Transaction, 10)
	for i := range txs {
		producers[i].NickName = fmt.Sprintf("Updated-%d", i)
		txs[i] = mockUpdateProducerTx(producers[i])
	}
	state.ProcessBlock(mockBlock(11, txs...), nil, false, 0)
	for i := range txs {
		p := state.GetProducer(producers[i].NodePublicKey)
		if !assert.Equal(t, fmt.Sprintf("Updated-%d", i), p.Info().NickName) {
			t.FailNow()
		}
	}

	// Cancel 10 producers.
	txs = make([]interfaces.Transaction, 10)
	for i := range txs {
		txs[i] = mockCancelProducerTx(producers[i].OwnerPublicKey)
	}
	state.ProcessBlock(mockBlock(12, txs...), nil, false, 0)
	// at this point, we have 10 canceled, 30 pending, 60 active and 90 in total producers.
	if !assert.Equal(t, 10, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 30, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 60, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 90, len(state.GetProducers())) {
		t.FailNow()
	}

	// Vote 10 producers for 10 times.
	publicKeys := make([][]byte, 10)
	for i, p := range producers[10:20] {
		publicKeys[i] = p.OwnerPublicKey
	}
	txs = make([]interfaces.Transaction, 10)
	for i := range txs {
		txs[i] = mockVoteTx(publicKeys)
	}
	state.ProcessBlock(mockBlock(13, txs...), nil, false, 0)
	for _, pk := range publicKeys {
		p := state.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(1000), p.Votes()) {
			t.FailNow()
		}
	}

	// Illegal 10 producers.
	txs = make([]interfaces.Transaction, 10)
	for i := range txs {
		txs[i] = mockIllegalBlockTx(producers[10+i].NodePublicKey)
	}
	state.ProcessBlock(mockBlock(14, txs...), nil, false, 0)
	// at this point, we have 10 canceled, 10 pending, 70 active, 10 illegal and 80 in total producers.
	if !assert.Equal(t, 10, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 70, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 80, len(state.GetProducers())) {
		t.FailNow()
	}

	// Mixed transactions 1 register, 2 cancel, 3 updates, 4 votes, 5 illegals.
	txs = make([]interfaces.Transaction, 15)
	info := &payload.ProducerInfo{
		OwnerPublicKey: randomPublicKey(),
		NodePublicKey:  make([]byte, 33),
	}
	rand.Read(info.NodePublicKey)
	info.NickName = "Producer-101"
	txs[0] = mockRegisterProducerTx(info)

	for i := 0; i < 2; i++ {
		txs[1+i] = mockCancelProducerTx(producers[20+i].OwnerPublicKey)
	}

	for i := 0; i < 3; i++ {
		txs[3+i] = mockUpdateProducerTx(producers[30+i])
	}

	publicKeys = make([][]byte, 4)
	for i, p := range producers[40:44] {
		publicKeys[i] = p.OwnerPublicKey
	}
	for i := 0; i < 4; i++ {
		txs[6+i] = mockVoteTx(publicKeys)
	}

	for i := 0; i < 5; i++ {
		txs[10+i] = mockIllegalBlockTx(producers[50+i].NodePublicKey)
	}
	state.ProcessBlock(mockBlock(15, txs...), nil, false, 0)
	// at this point, we have 12 canceled, 1 pending, 73 active, 15 illegal and 74 in total producers.
	// 10+2
	if !assert.Equal(t, 12, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	// 20-10+1
	if !assert.Equal(t, 1, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	// 60+10-2-5
	if !assert.Equal(t, 73, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	// 10+5
	if !assert.Equal(t, 15, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	// 101-12-15
	if !assert.Equal(t, 74, len(state.GetProducers())) {
		t.FailNow()
	}
	for _, pk := range publicKeys {
		p := state.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(400), p.Votes()) {
			t.FailNow()
		}
	}
}

func TestState_ProcessIllegalBlockEvidence(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.

	// Make producer 0 illegal.
	tx := mockIllegalBlockTx(producers[0].NodePublicKey)
	state.ProcessSpecialTxPayload(tx.Payload(), uint32(len(producers))+1)
	// At this point, we have 5 pending, 4 active 1 illegal and 9 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 4, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}

	// GetProcessor next height, state will rollback illegal producer.
	state.ProcessBlock(mockBlock(11), nil, false, 0)
	// At this point, we have 4 pending, 6 active and 10 in total producers.
	if !assert.Equal(t, 4, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_ProcessEmergencyInactiveArbitrators(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.

	// Make producer 0 illegal.
	tx := mockInactiveArbitratorsTx(producers[0].NodePublicKey)
	state.ProcessSpecialTxPayload(tx.Payload(), uint32(len(producers))+1)
	// At this point, we have 5 pending, 4 active 1 illegal and 9 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 4, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 0, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}

	// GetProcessor next height, state will rollback illegal producer.
	state.ProcessBlock(mockBlock(11), nil, false, 0)
	// At this point, we have 4 pending, 6 active and 10 in total producers.
	if !assert.Equal(t, 4, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_Rollback(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	err := state.RollbackTo(9)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	// At this point, we have 5 pending, 4 active and 9 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 4, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}
}

func TestState_GetHistory(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}
	// At this point, we have 5 pending, 5 active and 10 in total producers.

	// Test update producer.
	producers[0].NickName = "Updated"
	nodePublicKey := make([]byte, 33)
	rand.Read(nodePublicKey)
	producers[0].NodePublicKey = nodePublicKey
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil, false, 0)
	p := state.GetProducer(producers[0].NodePublicKey)
	if !assert.NotNil(t, p) {
		t.FailNow()
	}
	if !assert.Equal(t, "Updated", p.Info().NickName) {
		t.FailNow()
	}

	// Test cancel producer.
	tx = mockCancelProducerTx(producers[0].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil, false, 0)
	// At this point, we have 1 canceled, 3 pending, 6 active and 9 in total producers.

	// Test vote producer.
	publicKeys := make([][]byte, 5)
	for i, p := range producers[1:6] {
		publicKeys[i] = p.OwnerPublicKey
	}
	tx = mockVoteTx(publicKeys)
	state.ProcessBlock(mockBlock(13, tx), nil, false, 0)
	for _, pk := range publicKeys {
		p := state.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(100), p.Votes()) {
			t.FailNow()
		}
	}

	// Test illegal producer.
	tx = mockIllegalBlockTx(producers[1].NodePublicKey)
	state.ProcessBlock(mockBlock(14, tx), nil, false, 0)
	// At this point, we have 1 canceled, 1 pending, 7 active, 1 illegal and 8 in total producers.

	//_, err := state.GetHistory(0)
	//limitHeight := state.History.Height() - uint32(len(state.History.Changes()))
	//if !assert.EqualError(t, err, fmt.Sprintf("seek to %d overflow"+
	//	" History capacity, at most seek to %d", 0, limitHeight)) {
	//	t.FailNow()
	//}

	s, err := state.GetHistory(10)
	state.StateKeyFrame = s
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// At this point, we have 5 pending and 5 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	s, err = state.GetHistory(14)
	state.StateKeyFrame = s
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// At this point, we have 1 canceled, 1 pending, 7 active, 1 illegal and 8 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 7, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetProducers())) {
		t.FailNow()
	}

	s, err = state.GetHistory(12)
	state.StateKeyFrame = s
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	// At this point, we have 1 canceled, 3 pending, 6 active and 9 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 3, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 6, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 9, len(state.GetProducers())) {
		t.FailNow()
	}

	// GetProcessor a new height see if state go to best height.
	state.ProcessBlock(mockBlock(15, tx), nil, false, 0)
	// At this point, we have 1 canceled, 0 pending, 8 active, 1 illegal and 8 in total producers.
	if !assert.Equal(t, 1, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 0, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 1, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 8, len(state.GetProducers())) {
		t.FailNow()
	}

}

func TestState_NicknameExists(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	for i := range producers {
		if !assert.Equal(t, true, state.NicknameExists(
			fmt.Sprintf("Producer-%d", i+1))) {
			t.FailNow()
		}
	}

	// Change producer-1 nickname to Updated.
	producers[0].NickName = "Updated"
	nodePublicKey := make([]byte, 33)
	rand.Read(nodePublicKey)
	producers[0].NodePublicKey = nodePublicKey
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil, false, 0)
	p := state.GetProducer(producers[0].NodePublicKey)
	if !assert.NotNil(t, p) {
		t.FailNow()
	}
	if !assert.Equal(t, "Updated", p.Info().NickName) {
		t.FailNow()
	}

	if !assert.Equal(t, false, state.NicknameExists("Producer-1")) {
		t.FailNow()
	}

	// Cancel producer-2, see if nickname change to unused.
	tx = mockCancelProducerTx(producers[1].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil, false, 0)
	if !assert.Equal(t, false, state.NicknameExists("Producer-2")) {
		t.FailNow()
	}

	// Make producer-3 illegal, see if nickname change to unused.
	// Note: illegal is allowed to be activated now, so no need to remove nickname
	tx = mockIllegalBlockTx(producers[2].NodePublicKey)
	state.ProcessSpecialTxPayload(tx.Payload(), 13)
	if !assert.Equal(t, true, state.NicknameExists("Producer-3")) {
		t.FailNow()
	}
}

func TestState_ProducerExists(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	for _, p := range producers {
		if !assert.Equal(t, true, state.ProducerExists(p.NodePublicKey)) {
			t.FailNow()
		}
		if !assert.Equal(t, true, state.ProducerExists(p.OwnerPublicKey)) {
			t.FailNow()
		}
	}

	// Change producer node public key
	oldPublicKey := producers[0].NodePublicKey
	producers[0].NodePublicKey = make([]byte, 33)
	rand.Read(producers[0].NodePublicKey)
	tx := mockUpdateProducerTx(producers[0])
	state.ProcessBlock(mockBlock(11, tx), nil, false, 0)
	if !assert.Equal(t, true, state.ProducerExists(producers[0].NodePublicKey)) {
		t.FailNow()
	}
	if !assert.Equal(t, false, state.ProducerExists(oldPublicKey)) {
		t.FailNow()
	}

	// Canceled producer also existed.
	tx = mockCancelProducerTx(producers[0].OwnerPublicKey)
	state.ProcessBlock(mockBlock(12, tx), nil, false, 0)
	if !assert.Equal(t, true, state.ProducerExists(producers[0].OwnerPublicKey)) {
		t.FailNow()
	}
}

func TestState_IsDPOSTransaction(t *testing.T) {
	references := make(map[*common2.Input]common2.Output)
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
	state.GetTxReference = func(tx interfaces.Transaction) (
		map[*common2.Input]common2.Output, error) {
		return references, nil
	}

	producer := &payload.ProducerInfo{
		OwnerPublicKey: randomPublicKey(),
		NodePublicKey:  make([]byte, 33),
		NickName:       "Producer",
	}
	rand.Read(producer.NodePublicKey)

	tx := mockRegisterProducerTx(producer)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}
	state.ProcessBlock(mockBlock(1, tx), nil, false, 0)
	for i := uint32(2); i < 10; i++ {
		state.ProcessBlock(mockBlock(i), nil, false, 0)
	}

	tx = mockUpdateProducerTx(producer)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}

	tx = mockCancelProducerTx(producer.OwnerPublicKey)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}

	tx = mockVoteTx([][]byte{producer.OwnerPublicKey})
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx)) {
		t.FailNow()
	}
	state.ProcessBlock(mockBlock(10, tx), nil, false, 0)
	p := state.GetProducer(producer.NodePublicKey)
	if !assert.Equal(t, common.Fixed64(100), p.Votes()) {
		t.FailNow()
	}
	tx2 := mockCancelVoteTx(tx)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx2)) {
		t.FailNow()
	}
	for i, input := range tx2.Inputs() {
		references[input] = *tx.Outputs()[i]
	}
	state.ProcessBlock(mockBlock(11, tx2), nil, false, 0)
	p = state.GetProducer(producer.OwnerPublicKey)
	if !assert.Equal(t, common.Fixed64(0), p.Votes()) {
		t.FailNow()
	}

	tx2 = mockIllegalBlockTx(producer.OwnerPublicKey)
	if !assert.Equal(t, true, state.IsDPOSTransaction(tx2)) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_Normal(t *testing.T) {
	arbitrators := &state2.ArbitratorsMock{}
	state := state2.NewState(&config.DefaultParams, arbitrators.GetArbitrators, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
	state.ChainParams.InactivePenalty = 50
	state.ChainParams.ChangeCommitteeNewCRHeight = 1
	state.ChainParams.DPoSV2StartHeight = 1
	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  randomPublicKey(),
		}
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// Arbiters should set inactive after continuous three blocks
	ar1, _ := state2.NewOriginArbiter(producers[0].NodePublicKey)
	ar2, _ := state2.NewOriginArbiter(producers[1].NodePublicKey)
	ar3, _ := state2.NewOriginArbiter(producers[2].NodePublicKey)
	ar4, _ := state2.NewOriginArbiter(producers[3].NodePublicKey)
	ar5, _ := state2.NewOriginArbiter(producers[4].NodePublicKey)
	arbitrators.CurrentArbitrators = []state2.ArbiterMember{
		ar1, ar2, ar3, ar4, ar5,
	}

	currentHeight := 11
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for continuous 11 blocks
	for round := 0; round < 3; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {
			state.ProcessBlock(mockBlock(uint32(currentHeight)),
				&payload.Confirm{
					Proposal: payload.DPOSProposal{
						Sponsor: producers[arIndex].NodePublicKey,
					},
				}, false, 0)
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) ||
		!assert.True(t, state.IsInactiveProducer(producers[0].NodePublicKey)) {
		t.FailNow()
	}

	// check penalty
	inactiveProducer := state.GetProducer(producers[0].NodePublicKey)
	if !assert.Equal(t, inactiveProducer.Penalty(),
		state.ChainParams.InactivePenalty) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_FailNoContinuous(t *testing.T) {
	arbitrators := &state2.ArbitratorsMock{}
	state := state2.NewState(&config.DefaultParams, arbitrators.GetArbitrators, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  randomPublicKey(),
		}
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// Arbiters should set inactive after continuous three blocks
	ar1, _ := state2.NewOriginArbiter(producers[0].NodePublicKey)
	ar2, _ := state2.NewOriginArbiter(producers[1].NodePublicKey)
	ar3, _ := state2.NewOriginArbiter(producers[2].NodePublicKey)
	ar4, _ := state2.NewOriginArbiter(producers[3].NodePublicKey)
	ar5, _ := state2.NewOriginArbiter(producers[4].NodePublicKey)
	arbitrators.CurrentArbitrators = []state2.ArbiterMember{
		ar1, ar2, ar3, ar4, ar5,
	}

	currentHeight := 11
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for over 10 blocks,
	// but is not continuous
	for round := 0; round < 4; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {

			if round == 2 && arIndex == 4 {
				state.ProcessBlock(mockBlock(uint32(currentHeight)),
					&payload.Confirm{
						Proposal: payload.DPOSProposal{
							Sponsor: producers[0].NodePublicKey,
						},
					}, false, 0)
			} else {
				state.ProcessBlock(mockBlock(uint32(currentHeight)),
					&payload.Confirm{
						Proposal: payload.DPOSProposal{
							Sponsor: producers[arIndex].NodePublicKey,
						},
					}, false, 0)
			}
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_RecoverFromInactiveState(t *testing.T) {
	arbitrators := &state2.ArbitratorsMock{}
	state := state2.NewState(&config.DefaultParams, arbitrators.GetArbitrators, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
	state.ChainParams.ChangeCommitteeNewCRHeight = 1
	state.ChainParams.DPoSV2StartHeight = 1
	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  randomPublicKey(),
		}
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	// Arbiters should set inactive after continuous three blocks
	ar1, _ := state2.NewOriginArbiter(producers[0].NodePublicKey)
	ar2, _ := state2.NewOriginArbiter(producers[1].NodePublicKey)
	ar3, _ := state2.NewOriginArbiter(producers[2].NodePublicKey)
	ar4, _ := state2.NewOriginArbiter(producers[3].NodePublicKey)
	ar5, _ := state2.NewOriginArbiter(producers[4].NodePublicKey)
	arbitrators.CurrentArbitrators = []state2.ArbiterMember{
		ar1, ar2, ar3, ar4, ar5,
	}

	currentHeight := 11
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for continuous 11 blocks
	for round := 0; round < 3; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {
			state.ProcessBlock(mockBlock(uint32(currentHeight)),
				&payload.Confirm{
					Proposal: payload.DPOSProposal{
						Sponsor: producers[arIndex].NodePublicKey,
					},
				}, false, 0)
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) ||
		!assert.True(t, state.IsInactiveProducer(producers[0].NodePublicKey)) {
		t.FailNow()
	}

	// check penalty
	inactiveProducer := state.GetProducer(producers[0].NodePublicKey)
	if !assert.Equal(t, inactiveProducer.Penalty(),
		state.ChainParams.InactivePenalty) {
		t.FailNow()
	}

	// request for activating
	state.ProcessBlock(mockBlock(uint32(currentHeight),
		mockActivateProducerTx(producers[0].OwnerPublicKey)), nil, false, 0)
	currentHeight++

	// producer[0] will not be active util 6 blocks later
	for i := 0; i < 4; i++ {
		state.ProcessBlock(mockBlock(uint32(currentHeight)), nil, false, 0)
		if !assert.Equal(t, 1, len(state.GetInactiveProducers())) {
			t.FailNow()
		}
		currentHeight++
	}
	state.ProcessBlock(mockBlock(uint32(currentHeight)), nil, false, 0)
	if !assert.Equal(t, 0, len(state.GetInactiveProducers())) {
		t.FailNow()
	}
}

func TestState_InactiveProducer_DuringUpdateVersion(t *testing.T) {
	arbitrators := &state2.ArbitratorsMock{}
	state := state2.NewState(&config.DefaultParams, arbitrators.GetArbitrators, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
	state.ChainParams.InactivePenalty = 50

	// Create 10 producers info.
	producers := make([]*payload.ProducerInfo, 10)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  randomPublicKey(),
		}
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register each producer on one height.
	for i, p := range producers {
		tx := mockRegisterProducerTx(p)
		state.ProcessBlock(mockBlock(uint32(i+1), tx), nil, false, 0)
	}

	// At this point, we have 5 pending, 5 active and 10 in total producers.
	if !assert.Equal(t, 5, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 5, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetProducers())) {
		t.FailNow()
	}

	currentHeight := 11

	tx := functions.CreateTransaction(
		0,
		common2.UpdateVersion,
		0,
		&payload.UpdateVersion{
			StartHeight: uint32(currentHeight) + 1,
			EndHeight:   uint32(currentHeight) + 3*4 + 1,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	state.ProcessBlock(mockBlock(uint32(currentHeight), tx), nil, false, 0)

	// Arbiters should set inactive after continuous three blocks
	ar1, _ := state2.NewOriginArbiter(producers[0].NodePublicKey)
	ar2, _ := state2.NewOriginArbiter(producers[1].NodePublicKey)
	ar3, _ := state2.NewOriginArbiter(producers[2].NodePublicKey)
	ar4, _ := state2.NewOriginArbiter(producers[3].NodePublicKey)
	ar5, _ := state2.NewOriginArbiter(producers[4].NodePublicKey)
	arbitrators.CurrentArbitrators = []state2.ArbiterMember{
		ar1, ar2, ar3, ar4, ar5,
	}

	currentHeight++
	config.DefaultParams.PublicDPOSHeight = 11
	config.DefaultParams.MaxInactiveRounds = 10

	// simulate producers[0] do not sign for continuous 11 blocks
	for round := 0; round < 3; round++ {
		for arIndex := 1; arIndex <= 4; arIndex++ {
			state.ProcessBlock(mockBlock(uint32(currentHeight)),
				&payload.Confirm{
					Proposal: payload.DPOSProposal{
						Sponsor: producers[arIndex].NodePublicKey,
					},
				}, false, 0)
			currentHeight++
		}
	}

	// only producer[0] will be inactive
	if !assert.Equal(t, 1, len(state.GetInactiveProducers())) ||
		!assert.True(t, state.IsInactiveProducer(producers[0].NodePublicKey)) {
		t.FailNow()
	}

	// check penalty
	inactiveProducer := state.GetProducer(producers[0].NodePublicKey)
	if !assert.Equal(t, inactiveProducer.Penalty(), common.Fixed64(0)) {
		t.FailNow()
	}
}

func TestDPoSState_ProcessBlock_DepositAndReturnDeposit(t *testing.T) {
	arbitrators := &state2.ArbitratorsMock{}
	state := state2.NewState(&config.DefaultParams, arbitrators.GetArbitrators, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)
	height := config.DefaultParams.CRVotingStartHeight - 1

	_, pk, _ := crypto.GenerateKeyPair()
	pkBuf, _ := pk.EncodePoint(true)
	cont, _ := contract.CreateStandardContract(pk)
	depositCont, _ := contract.CreateDepositContractByPubKey(pk)

	// register register cr before CRVotingStartHeight
	registerTx := functions.CreateTransaction(
		0,
		common2.RegisterProducer,
		0,
		&payload.ProducerInfo{
			OwnerPublicKey: pkBuf,
			NodePublicKey:  pkBuf,
			NickName:       randomString(),
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(100),
			},
		},
		0,
		[]*program.Program{},
	)

	state.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: height,
		},
		Transactions: []interfaces.Transaction{registerTx},
	}, nil, false, 0)
	height++
	candidate := state.GetProducer(pkBuf)
	assert.Equal(t, common.Fixed64(100), candidate.TotalAmount())

	state.GetProducerDepositAmount = func(p common.Uint168) (
		fixed64 common.Fixed64, e error) {
		return common.Fixed64(100), nil
	}
	state.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: height,
		},
		Transactions: []interfaces.Transaction{},
	}, nil, false, 0)
	height++
	assert.Equal(t, common.Fixed64(100), candidate.TotalAmount())
	assert.Equal(t, state2.Pending, candidate.State())

	// deposit though normal tx
	tranferTx := functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		&payload.TransferAsset{},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(200),
			},
		},
		0,
		[]*program.Program{},
	)

	state.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: height,
		},
		Transactions: []interfaces.Transaction{tranferTx},
	}, nil, false, 0)
	height++
	assert.Equal(t, common.Fixed64(300), candidate.TotalAmount())

	// cancel candidate
	for i := 0; i < 4; i++ {
		state.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: height,
			},
			Transactions: []interfaces.Transaction{},
		}, nil, false, 0)
		height++
	}
	assert.Equal(t, state2.Active, candidate.State())

	var txs []interfaces.Transaction
	tx := functions.CreateTransaction(
		0,
		common2.CancelProducer,
		0,
		&payload.ProcessProducer{
			OwnerPublicKey: pkBuf,
		},
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txs = append(txs, tx)
	state.ProcessBlock(&types.Block{
		Header: common2.Header{
			Height: height,
		},
		Transactions: txs,
	}, nil, false, 0)
	height++
	for i := 0; i < 5; i++ {
		state.ProcessBlock(&types.Block{
			Header: common2.Header{
				Height: height,
			},
			Transactions: []interfaces.Transaction{},
		}, nil, false, 0)
		height++
	}
	assert.Equal(t, state2.Canceled, candidate.State())

	// return deposit
	tx = functions.CreateTransaction(
		0,
		common2.TransferAsset,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{
			{
				Previous: common2.OutPoint{
					TxID:  tranferTx.Hash(),
					Index: 0,
				},
			},
		},
		[]*common2.Output{},
		0,
		[]*program.Program{
			{
				Code: cont.Code,
			},
		},
	)

	state.ReturnDeposit(tx, height)
	state.History.Commit(height)
	assert.Equal(t, common.Fixed64(100), candidate.TotalAmount())
}

func TestState_CountArbitratorsInactivityV1(t *testing.T) {
	state := state2.NewState(&config.DefaultParams, nil, nil,
		func() bool { return false },
		nil, nil, nil,
		nil, nil, nil, nil)

	// Create 100 producers info.
	producers := make([]*payload.ProducerInfo, 100)
	for i, p := range producers {
		p = &payload.ProducerInfo{
			OwnerPublicKey: randomPublicKey(),
			NodePublicKey:  make([]byte, 33),
		}
		rand.Read(p.NodePublicKey)
		p.NickName = fmt.Sprintf("Producer-%d", i+1)
		producers[i] = p
	}

	// Register 10 producers on one height.
	for i := 0; i < 10; i++ {
		txs := make([]interfaces.Transaction, 10)
		for i, p := range producers[i*10 : (i+1)*10] {
			txs[i] = mockRegisterProducerTx(p)
		}
		state.ProcessBlock(mockBlock(uint32(i+1), txs...), nil, false, 0)
	}
	// at this point, we have 50 pending, 50 active and 100 in total producers.
	if !assert.Equal(t, 50, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 50, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 100, len(state.GetProducers())) {
		t.FailNow()
	}

	// Update 10 producers.
	txs := make([]interfaces.Transaction, 10)
	for i := range txs {
		producers[i].NickName = fmt.Sprintf("Updated-%d", i)
		txs[i] = mockUpdateProducerTx(producers[i])
	}
	state.ProcessBlock(mockBlock(11, txs...), nil, false, 0)
	for i := range txs {
		p := state.GetProducer(producers[i].NodePublicKey)
		if !assert.Equal(t, fmt.Sprintf("Updated-%d", i), p.Info().NickName) {
			t.FailNow()
		}
	}

	// Cancel 10 producers.
	txs = make([]interfaces.Transaction, 10)
	for i := range txs {
		txs[i] = mockCancelProducerTx(producers[i].OwnerPublicKey)
	}
	state.ProcessBlock(mockBlock(12, txs...), nil, false, 0)
	// at this point, we have 10 canceled, 30 pending, 60 active and 90 in total producers.
	if !assert.Equal(t, 10, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 30, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 60, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 90, len(state.GetProducers())) {
		t.FailNow()
	}

	// Vote 10 producers for 10 times.
	publicKeys := make([][]byte, 10)
	for i, p := range producers[10:20] {
		publicKeys[i] = p.OwnerPublicKey
	}
	txs = make([]interfaces.Transaction, 10)
	for i := range txs {
		txs[i] = mockVoteTx(publicKeys)
	}
	state.ProcessBlock(mockBlock(13, txs...), nil, false, 0)
	for _, pk := range publicKeys {
		p := state.GetProducer(pk)
		if !assert.Equal(t, common.Fixed64(1000), p.Votes()) {
			t.FailNow()
		}
	}

	// Illegal 10 producers.
	txs = make([]interfaces.Transaction, 10)
	for i := range txs {
		txs[i] = mockIllegalBlockTx(producers[10+i].NodePublicKey)
	}
	state.ProcessBlock(mockBlock(14, txs...), nil, false, 0)
	// at this point, we have 10 canceled, 10 pending, 70 active, 10 illegal and 80 in total producers.
	if !assert.Equal(t, 10, len(state.GetCanceledProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetPendingProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 70, len(state.GetActiveProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 10, len(state.GetIllegalProducers())) {
		t.FailNow()
	}
	if !assert.Equal(t, 80, len(state.GetProducers())) {
		t.FailNow()
	}

	pds := state.GetActiveProducers()
	// register getArbiters function
	state.GetArbiters = func() []*state2.ArbiterInfo {
		result := make([]*state2.ArbiterInfo, 0)
		for i, p := range pds {
			if i >= len(state.ChainParams.CRCArbiters)+state.ChainParams.GeneralArbiters {
				break
			}
			result = append(result, &state2.ArbiterInfo{
				NodePublicKey: p.NodePublicKey(),
				IsNormal:      true,
			})
		}
		return result
	}

	// set the random DPOS node.
	pds[len(pds)-1].SetSelected(true)
	config.DefaultParams.MaxInactiveRounds = 36
	config.DefaultParams.MaxInactiveRoundsOfRandomNode = 10

	// random DPOS node does not work for 10 turns.
	for i := 0; i < 10*36; i++ {
		p := pds[i%36]
		nodePublcKey := p.NodePublicKey()
		if p.Selected() {
			nodePublcKey = pds[0].NodePublicKey()
		}
		height := state.ChainParams.ChangeCommitteeNewCRHeight + 1 + uint32(i)
		state.CountArbitratorsInactivityV1(
			height,
			&payload.Confirm{
				Proposal: payload.DPOSProposal{
					Sponsor: nodePublcKey,
				},
			})
		state.History.Commit(height)
	}

	// check the status of random DPOS node.
	for i, p := range pds {
		if i >= 36 {
			break
		}
		if p.Selected() {
			assert.Equal(t, state2.Inactive, p.State())
		} else {
			assert.Equal(t, state2.Active, p.State())
		}
	}
}
