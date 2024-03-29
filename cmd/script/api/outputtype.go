// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package api

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/log"
	"github.com/elastos/Elastos.ELA/core/contract"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/crypto"

	lua "github.com/yuin/gopher-lua"
)

const (
	luaOutputTypeName              = "output"
	luaVoteOutputTypeName          = "voteoutput"
	luaStakeOutputTypeName         = "stakeoutput"
	luaNewCrossChainOutputTypeName = "crosschainoutput"
	luaDefaultOutputTypeName       = "defaultoutput"
	luaVoteContentTypeName         = "votecontent"
)

func RegisterOutputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaOutputTypeName)
	L.SetGlobal("output", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newTxOutput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), outputMethods))
}

// Constructor
func newTxOutput(L *lua.LState) int {
	assetIDStr := L.ToString(1)
	value := L.ToInt64(2)
	address := L.ToString(3)
	outputType := L.ToInt(4)
	outputPayloadData := L.CheckUserData(5)

	assetIDSlice, _ := hex.DecodeString(assetIDStr)
	assetIDSlice = common.BytesReverse(assetIDSlice)
	var assetID common.Uint256
	copy(assetID[:], assetIDSlice[0:32])

	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var outputPayload common2.OutputPayload
	switch outputPayloadData.Value.(type) {
	case *outputpayload.DefaultOutput:
		payload, ok := outputPayloadData.Value.(*outputpayload.DefaultOutput)
		if !ok {
			log.Debug("error default output payload")
		}
		outputPayload = payload
	case *outputpayload.VoteOutput:
		payload, ok := outputPayloadData.Value.(*outputpayload.VoteOutput)
		if !ok {
			log.Debug("error vote output payload")
		}
		outputPayload = payload
	case *outputpayload.CrossChainOutput:
		payload, ok := outputPayloadData.Value.(*outputpayload.CrossChainOutput)
		if !ok {
			log.Debug("error cross chain payload")
		}
		outputPayload = payload

	case *outputpayload.ExchangeVotesOutput:
		payload, ok := outputPayloadData.Value.(*outputpayload.ExchangeVotesOutput)
		if !ok {
			log.Debug("error exchange vote payload")
		}
		outputPayload = payload
	}

	output := &common2.Output{
		AssetID:     assetID,
		Value:       common.Fixed64(value),
		OutputLock:  0,
		ProgramHash: *programHash,
		Type:        common2.OutputType(outputType),
		Payload:     outputPayload,
	}

	ud := L.NewUserData()
	ud.Value = output
	L.SetMetatable(ud, L.GetTypeMetatable(luaOutputTypeName))
	L.Push(ud)

	return 1
}

// Checks whether the first lua argument is a *LUserData with *Output and returns this *Output.
func checkTxOutput(L *lua.LState, idx int) *common2.Output {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*common2.Output); ok {
		return v
	}
	L.ArgError(1, "Output expected")
	return nil
}

var outputMethods = map[string]lua.LGFunction{
	"get": outputGet,
}

// Getter and setter for the Person#Name
func outputGet(L *lua.LState) int {
	p := checkTxOutput(L, 1)
	fmt.Println(p)

	return 0
}

// Default Output Payload
func RegisterDefaultOutputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaDefaultOutputTypeName)
	L.SetGlobal("defaultoutput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newDefaultOutput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), newDefaultOutputMethods))
}

func newDefaultOutput(L *lua.LState) int {
	defaultOutput := &outputpayload.DefaultOutput{}
	ud := L.NewUserData()
	ud.Value = defaultOutput
	L.SetMetatable(ud, L.GetTypeMetatable(luaDefaultOutputTypeName))
	L.Push(ud)

	return 1
}

func checkDefaultOutput(L *lua.LState, idx int) *outputpayload.DefaultOutput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*outputpayload.DefaultOutput); ok {
		return v
	}
	L.ArgError(1, "OTNone expected")
	return nil
}

var newDefaultOutputMethods = map[string]lua.LGFunction{
	"get": defaultOutputGet,
}

func defaultOutputGet(L *lua.LState) int {
	p := checkDefaultOutput(L, 1)
	fmt.Println(p)

	return 0
}

// OTVote Output Payload
func RegisterVoteOutputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaVoteOutputTypeName)
	L.SetGlobal("voteoutput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newVoteOutput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), newVoteOutputMethods))
}

func newVoteOutput(L *lua.LState) int {
	version := L.ToInt(1)
	contentsTable := L.ToTable(2)

	contents := make([]outputpayload.VoteContent, 0)
	contentsTable.ForEach(func(i, v lua.LValue) {
		lv, ok := v.(*lua.LUserData)
		if !ok {
			println("error vote content user data")
		}
		content, ok := lv.Value.(*outputpayload.VoteContent)
		if !ok {
			fmt.Println("error vote content")
		}
		contents = append(contents, *content)
	})

	voteOutput := &outputpayload.VoteOutput{
		Version:  byte(version),
		Contents: contents,
	}
	ud := L.NewUserData()
	ud.Value = voteOutput
	L.SetMetatable(ud, L.GetTypeMetatable(luaVoteOutputTypeName))
	L.Push(ud)

	return 1
}

func checkVoteOutput(L *lua.LState, idx int) *outputpayload.VoteOutput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*outputpayload.VoteOutput); ok {
		return v
	}
	L.ArgError(1, "OTVote expected")
	return nil
}

var newVoteOutputMethods = map[string]lua.LGFunction{
	"get": voteOutputGet,
}

func voteOutputGet(L *lua.LState) int {
	p := checkVoteOutput(L, 1)
	fmt.Println(p)

	return 0
}

// OTVote Output Payload
func RegisterStakeOutputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaStakeOutputTypeName)
	L.SetGlobal("stakeoutput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newStakeOutput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), newStakeOutputMethods))
}

func newStakeOutput(L *lua.LState) int {
	version := L.ToInt(1)
	address := L.ToString(2)

	programHash, err := common.Uint168FromAddress(address)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	voteOutput := &outputpayload.ExchangeVotesOutput{
		Version:      byte(version),
		StakeAddress: *programHash,
	}
	ud := L.NewUserData()
	ud.Value = voteOutput
	L.SetMetatable(ud, L.GetTypeMetatable(luaStakeOutputTypeName))
	L.Push(ud)

	return 1
}

func checkStakeOutput(L *lua.LState, idx int) *outputpayload.ExchangeVotesOutput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*outputpayload.ExchangeVotesOutput); ok {
		return v
	}
	L.ArgError(1, "ExchangeVotesOutput expected")
	return nil
}

var newStakeOutputMethods = map[string]lua.LGFunction{
	"get": stakeOutputGet,
}

func stakeOutputGet(L *lua.LState) int {
	p := checkStakeOutput(L, 1)
	fmt.Println(p)

	return 0
}

// OTVote Content
func RegisterVoteContentType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaVoteContentTypeName)
	L.SetGlobal("votecontent", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newVoteContent))
	L.SetField(mt, "newcr", L.NewFunction(newVoteCRContent))
	L.SetField(mt, "newdposv2", L.NewFunction(newVoteDposV2Content))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), newVoteContentMethods))
}

func newVoteContent(L *lua.LState) int {
	voteType := L.ToInt(1)
	candidatesTable := L.ToTable(2)
	candidateVotesTable := L.ToTable(3)

	candidates := make([][]byte, 0)
	votes := make([]common.Fixed64, 0)
	candidatesTable.ForEach(func(i, value lua.LValue) {
		publicKey := lua.LVAsString(value)
		publicKey = strings.Replace(publicKey, "{", "", 1)
		publicKey = strings.Replace(publicKey, "}", "", 1)
		pk, err := common.HexStringToBytes(publicKey)
		if err != nil {
			fmt.Println("invalid public key")
			os.Exit(1)
		}
		candidates = append(candidates, pk)
	})
	candidateVotesTable.ForEach(func(i, value lua.LValue) {
		voteStr := lua.LVAsString(value)
		voteStr = strings.Replace(voteStr, "{", "", 1)
		voteStr = strings.Replace(voteStr, "}", "", 1)
		vote, err := strconv.ParseFloat(voteStr, 64)
		if err != nil {
			fmt.Println("invalid votes")
			os.Exit(1)
		}
		votes = append(votes, common.Fixed64(int64(vote*1e8)))
	})

	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(candidates))
	for i := 0; i < len(candidates); i++ {
		candidateVotes = append(candidateVotes, outputpayload.CandidateVotes{
			Candidate: candidates[i],
			Votes:     votes[i],
		})
	}

	voteContent := &outputpayload.VoteContent{
		VoteType:       outputpayload.VoteType(voteType),
		CandidateVotes: candidateVotes,
	}

	ud := L.NewUserData()
	ud.Value = voteContent
	L.SetMetatable(ud, L.GetTypeMetatable(luaVoteContentTypeName))
	L.Push(ud)

	return 1
}

func newVoteCRContent(L *lua.LState) int {
	voteType := L.ToInt(1)
	candidatesTable := L.ToTable(2)
	candidateVotesTable := L.ToTable(3)

	candidates := make([][]byte, 0)
	votes := make([]common.Fixed64, 0)
	candidatesTable.ForEach(func(i, value lua.LValue) {
		publicKey := lua.LVAsString(value)
		publicKey = strings.Replace(publicKey, "{", "", 1)
		publicKey = strings.Replace(publicKey, "}", "", 1)
		pk, err := common.HexStringToBytes(publicKey)
		if err != nil {
			fmt.Println("invalid public key")
			os.Exit(1)
		}
		candidates = append(candidates, pk)
	})
	candidateVotesTable.ForEach(func(i, value lua.LValue) {
		voteStr := lua.LVAsString(value)
		voteStr = strings.Replace(voteStr, "{", "", 1)
		voteStr = strings.Replace(voteStr, "}", "", 1)
		vote, err := strconv.ParseFloat(voteStr, 64)
		if err != nil {
			fmt.Println("invalid votes")
			os.Exit(1)
		}
		votes = append(votes, common.Fixed64(int64(vote*1e8)))
	})

	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(candidates))
	for i := 0; i < len(candidates); i++ {
		pk, err := crypto.DecodePoint(candidates[i])
		if err != nil {
			fmt.Println("wrong cr public key")
			os.Exit(1)
		}
		code, err := contract.CreateStandardRedeemScript(pk)
		if err != nil {
			fmt.Println("wrong cr public key")
			os.Exit(1)
		}

		cidProgramHash := getIDProgramHash(code)
		candidateVotes = append(candidateVotes, outputpayload.CandidateVotes{
			Candidate: cidProgramHash.Bytes(),
			Votes:     votes[i],
		})
	}

	voteContent := &outputpayload.VoteContent{
		VoteType:       outputpayload.VoteType(voteType),
		CandidateVotes: candidateVotes,
	}

	ud := L.NewUserData()
	ud.Value = voteContent
	L.SetMetatable(ud, L.GetTypeMetatable(luaVoteContentTypeName))
	L.Push(ud)

	return 1
}

func newVoteDposV2Content(L *lua.LState) int {
	voteType := L.ToInt(1)
	candidatesTable := L.ToTable(2)
	candidateVotesTable := L.ToTable(3)

	candidates := make([][]byte, 0)
	votes := make([]common.Fixed64, 0)
	candidatesTable.ForEach(func(i, value lua.LValue) {
		publicKey := lua.LVAsString(value)
		publicKey = strings.Replace(publicKey, "{", "", 1)
		publicKey = strings.Replace(publicKey, "}", "", 1)
		pk, err := common.HexStringToBytes(publicKey)
		if err != nil {
			fmt.Println("invalid public key")
			os.Exit(1)
		}
		candidates = append(candidates, pk)
	})
	candidateVotesTable.ForEach(func(i, value lua.LValue) {
		voteStr := lua.LVAsString(value)
		voteStr = strings.Replace(voteStr, "{", "", 1)
		voteStr = strings.Replace(voteStr, "}", "", 1)
		vote, err := strconv.ParseFloat(voteStr, 64)
		if err != nil {
			fmt.Println("invalid votes")
			os.Exit(1)
		}
		votes = append(votes, common.Fixed64(int64(vote*1e8)))
	})

	candidateVotes := make([]outputpayload.CandidateVotes, 0, len(candidates))
	for i := 0; i < len(candidates); i++ {
		candidateVotes = append(candidateVotes, outputpayload.CandidateVotes{
			Candidate: candidates[i],
			Votes:     votes[i],
		})
	}

	voteContent := &outputpayload.VoteContent{
		VoteType:       outputpayload.VoteType(voteType),
		CandidateVotes: candidateVotes,
	}

	ud := L.NewUserData()
	ud.Value = voteContent
	L.SetMetatable(ud, L.GetTypeMetatable(luaVoteContentTypeName))
	L.Push(ud)

	return 1
}

func checkVoteContent(L *lua.LState, idx int) *outputpayload.VoteContent {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*outputpayload.VoteContent); ok {
		return v
	}
	L.ArgError(1, "Vote expected")
	return nil
}

var newVoteContentMethods = map[string]lua.LGFunction{
	"get": voteContentGet,
}

func voteContentGet(L *lua.LState) int {
	p := checkVoteContent(L, 1)
	fmt.Println(p)

	return 0
}

// New cross chain Content
func RegisterNewCrossChainOutputType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaNewCrossChainOutputTypeName)
	L.SetGlobal("crosschainoutput", mt)
	// static attributes
	L.SetField(mt, "new", L.NewFunction(newCrossChainOutput))
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), newCrossChainOutputMethods))
}

func newCrossChainOutput(L *lua.LState) int {
	targetAddress := L.ToString(1)
	amount := L.ToInt(2)
	targetData := L.ToString(3)

	var output = &outputpayload.CrossChainOutput{
		Version:       0,
		TargetAddress: targetAddress,
		TargetAmount:  common.Fixed64(amount),
		TargetData:    []byte(targetData),
	}
	ud := L.NewUserData()
	ud.Value = output
	L.SetMetatable(ud, L.GetTypeMetatable(luaNewCrossChainOutputTypeName))
	L.Push(ud)

	return 1
}

func checkNewCrossChainOutput(L *lua.LState, idx int) *outputpayload.CrossChainOutput {
	ud := L.CheckUserData(idx)
	if v, ok := ud.Value.(*outputpayload.CrossChainOutput); ok {
		return v
	}
	L.ArgError(1, "New CrossChainOutput expected")
	return nil
}

var newCrossChainOutputMethods = map[string]lua.LGFunction{
	"get": crossChainOutputGet,
}

func crossChainOutputGet(L *lua.LState) int {
	p := checkNewCrossChainOutput(L, 1)
	fmt.Println(p)

	return 0
}
