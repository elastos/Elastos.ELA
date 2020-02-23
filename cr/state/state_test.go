// Copyright (c) 2017-2019 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package state

import (
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/utils"

	"github.com/stretchr/testify/assert"
)

func TestState_GetCandidatesRelated(t *testing.T) {
	keyFrame := *randomStateKeyFrame(5, true)
	state := State{
		StateKeyFrame: keyFrame,
	}

	// get single candidate
	for k, v := range keyFrame.PendingCandidates {
		v2 := state.GetCandidateByCID(k)
		assert.True(t, candidateEqual(v, v2))

		v3 := state.GetCandidate(v.info.Code)
		assert.True(t, candidateEqual(v, v3))
	}
	for k, v := range keyFrame.ActivityCandidates {
		v2 := state.GetCandidateByCID(k)
		assert.True(t, candidateEqual(v, v2))

		v3 := state.GetCandidate(v.info.Code)
		assert.True(t, candidateEqual(v, v3))
	}
	for k, v := range keyFrame.CanceledCandidates {
		v2 := state.GetCandidateByCID(k)
		assert.True(t, candidateEqual(v, v2))

		v3 := state.GetCandidate(v.info.Code)
		assert.True(t, candidateEqual(v, v3))
	}

	// get candidates
	candidates := state.GetAllCandidates()
	assert.Equal(t, 15, len(candidates))

	pending := state.GetCandidates(Pending)
	assert.Equal(t, 5, len(pending))

	actives := state.GetCandidates(Active)
	assert.Equal(t, 5, len(actives))

	cancels := state.GetCandidates(Canceled)
	assert.Equal(t, 3, len(cancels))

	returns := state.GetCandidates(Returned)
	assert.Equal(t, 2, len(returns))
}

func TestState_ExistCandidateRelated(t *testing.T) {
	keyFrame := *randomStateKeyFrame(5, true)
	state := State{
		StateKeyFrame: keyFrame,
	}

	assert.False(t, state.ExistCandidate(make([]byte, 34)))
	assert.False(t, state.ExistCandidateByCID(common.Uint168{}))
	assert.False(t, state.ExistCandidateByNickname(""))

	for _, v := range keyFrame.PendingCandidates {
		assert.True(t, state.ExistCandidate(v.info.Code))
		assert.True(t, state.ExistCandidateByCID(v.info.CID))
		assert.True(t, state.ExistCandidateByNickname(v.info.NickName))
	}

	for _, v := range keyFrame.ActivityCandidates {
		assert.True(t, state.ExistCandidate(v.info.Code))
		assert.True(t, state.ExistCandidateByCID(v.info.CID))
		assert.True(t, state.ExistCandidateByNickname(v.info.NickName))
	}

	for _, v := range keyFrame.CanceledCandidates {
		assert.True(t, state.ExistCandidate(v.info.Code))
		assert.True(t, state.ExistCandidateByCID(v.info.CID))
		assert.True(t, state.ExistCandidateByNickname(v.info.NickName))
	}
}

func getCode(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}

func TestState_ProcessBlock_PendingUpdateThenCancel(t *testing.T) {
	state := NewState(nil)
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	code := getCode(publicKeyStr1)
	cid := *getCID(code)
	nickname := randomString()

	assert.False(t, state.ExistCandidate(code))
	assert.False(t, state.ExistCandidateByCID(cid))
	assert.False(t, state.ExistCandidateByNickname(nickname))

	// register CR
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: 1,
		},
		Transactions: []*types.Transaction{
			generateRegisterCR(code, cid, nickname),
		},
	}, nil)
	assert.True(t, state.ExistCandidate(code))
	assert.True(t, state.ExistCandidateByCID(cid))
	assert.True(t, state.ExistCandidateByNickname(nickname))
	candidate := state.GetCandidate(code)
	assert.Equal(t, Pending, candidate.state)

	// update pending CR
	nickname2 := randomString()
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: 2,
		},
		Transactions: []*types.Transaction{
			generateUpdateCR(code, cid, nickname2),
		},
	}, nil)
	assert.True(t, state.ExistCandidate(code))
	assert.True(t, state.ExistCandidateByCID(cid))
	assert.False(t, state.ExistCandidateByNickname(nickname))
	assert.True(t, state.ExistCandidateByNickname(nickname2))
	candidate = state.GetCandidate(code)
	assert.Equal(t, Pending, candidate.state)

	//cancel pending CR
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: 3,
		},
		Transactions: []*types.Transaction{
			generateUnregisterCR(code),
		},
	}, nil)
	assert.True(t, state.ExistCandidate(code))
	assert.True(t, state.ExistCandidateByCID(cid))
	assert.False(t, state.ExistCandidateByNickname(nickname))
	assert.False(t, state.ExistCandidateByNickname(nickname2))
	candidate = state.GetCandidate(code)
	assert.Equal(t, Canceled, candidate.state)
	assert.Equal(t, 0, len(state.GetCandidates(Pending)))
	assert.Equal(t, 1, len(state.GetCandidates(Canceled)))
}

func TestState_ProcessBlock_PendingActiveThenCancel(t *testing.T) {
	state := NewState(nil)
	height := uint32(1)
	nickname := randomString()
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	code := getCode(publicKeyStr1)
	cid := *getCID(code)

	assert.False(t, state.ExistCandidate(code))
	assert.False(t, state.ExistCandidateByCID(cid))
	assert.False(t, state.ExistCandidateByNickname(nickname))

	// register CR
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{
			generateRegisterCR(code, cid, nickname),
		},
	}, nil)
	height++
	assert.True(t, state.ExistCandidate(code))
	assert.True(t, state.ExistCandidateByCID(cid))
	assert.True(t, state.ExistCandidateByNickname(nickname))
	candidate := state.GetCandidate(code)
	assert.Equal(t, Pending, candidate.state)

	// register CR then after 6 block should be active state
	for i := 0; i < 5; i++ {
		state.ProcessBlock(&types.Block{
			Header: types.Header{
				Height: height,
			},
			Transactions: []*types.Transaction{},
		}, nil)
		height++
	}
	candidate = state.GetCandidate(code)
	assert.Equal(t, Active, candidate.state)

	// update active CR
	nickname2 := randomString()
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{
			generateUpdateCR(code, cid, nickname2),
		},
	}, nil)
	height++
	assert.True(t, state.ExistCandidate(code))
	assert.True(t, state.ExistCandidateByCID(cid))
	assert.False(t, state.ExistCandidateByNickname(nickname))
	assert.True(t, state.ExistCandidateByNickname(nickname2))
	candidate = state.GetCandidate(code)
	assert.Equal(t, Active, candidate.state)

	// cancel active CR
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{
			generateUnregisterCR(code),
		},
	}, nil)
	assert.True(t, state.ExistCandidate(code))
	assert.True(t, state.ExistCandidateByCID(cid))
	assert.False(t, state.ExistCandidateByNickname(nickname))
	assert.False(t, state.ExistCandidateByNickname(nickname2))
	candidate = state.GetCandidate(code)
	assert.Equal(t, Canceled, candidate.state)
	assert.Equal(t, 0, len(state.GetCandidates(Pending)))
	assert.Equal(t, 1, len(state.GetCandidates(Canceled)))
}

func TestState_ProcessBlock_MixedCRProcessing(t *testing.T) {
	state := State{
		StateKeyFrame: *randomStateKeyFrame(5, true),
		history:       utils.NewHistory(maxHistoryCapacity),
	}
	height := uint32(1)

	assert.Equal(t, 15, len(state.GetAllCandidates()))
	assert.Equal(t, 5, len(state.GetCandidates(Pending)))
	assert.Equal(t, 5, len(state.GetCandidates(Active)))
	assert.Equal(t, 3, len(state.GetCandidates(Canceled)))
	assert.Equal(t, 2, len(state.GetCandidates(Returned)))

	for i := 0; i < 10; i++ {
		code := randomBytes(34)
		nickname := randomString()
		cid := *randomUint168()

		state.ProcessBlock(&types.Block{
			Header: types.Header{
				Height: height,
			},
			Transactions: []*types.Transaction{
				generateRegisterCR(code, cid, nickname),
			},
		}, nil)
		height++
	}
	assert.Equal(t, 25, len(state.GetAllCandidates()))
	assert.Equal(t, 5, len(state.GetCandidates(Pending)))
	assert.Equal(t, 15, len(state.GetCandidates(Active)))
	assert.Equal(t, 3, len(state.GetCandidates(Canceled)))
	assert.Equal(t, 2, len(state.GetCandidates(Returned)))

	for i := 0; i < 5; i++ {
		state.ProcessBlock(&types.Block{
			Header: types.Header{
				Height: height,
			},
			Transactions: []*types.Transaction{},
		}, nil)
		height++
	}
	assert.Equal(t, 25, len(state.GetAllCandidates()))
	assert.Equal(t, 0, len(state.GetCandidates(Pending)))
	assert.Equal(t, 20, len(state.GetCandidates(Active)))
	assert.Equal(t, 3, len(state.GetCandidates(Canceled)))
	assert.Equal(t, 2, len(state.GetCandidates(Returned)))
}

func TestState_ProcessBlock_VotingAndCancel(t *testing.T) {
	keyframe := randomStateKeyFrame(5, true)
	state := NewState(nil)
	state.StateKeyFrame = *keyframe
	state.history = utils.NewHistory(maxHistoryCapacity)
	height := uint32(1)

	activeCodes := make([][]byte, 0, 5)
	for _, v := range keyframe.ActivityCandidates {
		v.votes = 0
		activeCodes = append(activeCodes, v.info.Code)
	}

	// vote for the active candidates
	voteTx := mockNewVoteTx(activeCodes)
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{voteTx},
	}, nil)
	height++

	for i, v := range activeCodes {
		candidate := state.GetCandidate(v)
		assert.Equal(t, common.Fixed64((i+1)*10), candidate.votes)
	}

	// cancel votes the active candidates
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{
			{
				Inputs: []*types.Input{
					{
						Previous: *types.NewOutPoint(voteTx.Hash(), uint16(0)),
					},
				},
			},
		},
	}, nil)

	for _, v := range activeCodes {
		candidate := state.GetCandidate(v)
		assert.Equal(t, common.Fixed64(0), candidate.votes)
	}
}

func TestState_ProcessBlock_DepositAndReturnDeposit(t *testing.T) {
	state := NewState(nil)
	height := uint32(1)

	_, pk, _ := crypto.GenerateKeyPair()
	cont, _ := contract.CreateStandardContract(pk)
	code := cont.Code
	depositCont, _ := contract.CreateDepositContractByPubKey(pk)

	// register CR
	registerCRTx := &types.Transaction{
		TxType: types.RegisterCR,
		Payload: &payload.CRInfo{
			Code:     code,
			CID:      *getCID(code),
			NickName: randomString(),
		},
		Outputs: []*types.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(100),
			},
		},
	}
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{registerCRTx},
	}, nil)
	height++
	candidate := state.GetCandidate(code)
	assert.Equal(t, common.Fixed64(100), candidate.depositAmount)

	// deposit though normal tx
	tranferTx := &types.Transaction{
		TxType:  types.TransferAsset,
		Payload: &payload.TransferAsset{},
		Outputs: []*types.Output{
			{
				ProgramHash: *depositCont.ToProgramHash(),
				Value:       common.Fixed64(200),
			},
		},
	}
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{tranferTx},
	}, nil)
	height++
	assert.Equal(t, common.Fixed64(300), candidate.depositAmount)

	// cancel candidate
	for i := 0; i < 4; i++ {
		state.ProcessBlock(&types.Block{
			Header: types.Header{
				Height: height,
			},
			Transactions: []*types.Transaction{},
		}, nil)
		height++
	}
	assert.Equal(t, Active, candidate.state)
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{
			generateUnregisterCR(code),
		},
	}, nil)
	height++
	for i := 0; i < 5; i++ {
		state.ProcessBlock(&types.Block{
			Header: types.Header{
				Height: height,
			},
			Transactions: []*types.Transaction{},
		}, nil)
		height++
	}
	assert.Equal(t, Canceled, candidate.state)

	// return deposit
	rdTx := generateReturnCRDeposit(code)
	rdTx.Inputs = []*types.Input{
		{
			Previous: types.OutPoint{
				TxID:  registerCRTx.Hash(),
				Index: 0,
			},
		},
		{
			Previous: types.OutPoint{
				TxID:  tranferTx.Hash(),
				Index: 0,
			},
		},
	}
	state.ProcessBlock(&types.Block{
		Header: types.Header{
			Height: height,
		},
		Transactions: []*types.Transaction{rdTx},
	}, nil)
	state.history.Commit(height)
	assert.Equal(t, common.Fixed64(0), candidate.depositAmount)
}

func mockNewVoteTx(programCodes [][]byte) *types.Transaction {
	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(programCodes))
	for i, pk := range programCodes {
		//code := getCode(common.BytesToHexString(pk))
		cid := getCID(pk)
		candidateVotes = append(candidateVotes,
			outputpayload.CandidateVotes{
				Candidate: cid.Bytes(),
				Votes:     common.Fixed64((i + 1) * 10)})
	}
	output := &types.Output{
		Value: 100,
		Type:  types.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: outputpayload.VoteProducerAndCRVersion,
			Contents: []outputpayload.VoteContent{
				{outputpayload.CRC, candidateVotes},
			},
		},
	}

	return &types.Transaction{
		Version: types.TxVersion09,
		TxType:  types.TransferAsset,
		Outputs: []*types.Output{output},
	}
}

func generateRegisterCR(code []byte, cid common.Uint168,
	nickname string) *types.Transaction {
	return &types.Transaction{
		TxType: types.RegisterCR,
		Payload: &payload.CRInfo{
			Code:     code,
			CID:      cid,
			NickName: nickname,
		},
	}
}

func generateUpdateCR(code []byte, cid common.Uint168,
	nickname string) *types.Transaction {
	return &types.Transaction{
		TxType: types.UpdateCR,
		Payload: &payload.CRInfo{
			Code:     code,
			CID:      cid,
			NickName: nickname,
		},
	}
}

func generateUnregisterCR(code []byte) *types.Transaction {
	return &types.Transaction{
		TxType: types.UnregisterCR,
		Payload: &payload.UnregisterCR{
			CID: *getCID(code),
		},
	}
}

func getCID(code []byte) *common.Uint168 {
	ct1, _ := contract.CreateCRIDContractByCode(code)
	return ct1.ToProgramHash()
}

func generateReturnCRDeposit(code []byte) *types.Transaction {
	return &types.Transaction{
		TxType:  types.ReturnCRDepositCoin,
		Payload: &payload.ReturnDepositCoin{},
		Programs: []*program.Program{
			&program.Program{
				Code: code,
			},
		},
	}
}
