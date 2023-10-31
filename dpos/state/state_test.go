package state

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProducer_ActivateRequestHeight(t *testing.T) {
	detailedDPoSV2Votes := make(map[common.Uint168]map[common.Uint256]payload.DetailedVoteInfo)
	programHash1, _ := common.Uint168FromAddress("SWqLCTLHCs7pMzPwUU9CK2zeMceb6XzG4v")
	txhash1, _ := common.Uint256FromHexString("f201c6b8eec0abf24d66881b761bff1b18782deed06e5cafa8983b24a679e4e8")
	candidate1, _ := hex.DecodeString("03878cbe6abdafc702befd90e2329c4f37e7cb166410f0ecb70488c74c85b81d66")
	var Info []payload.VotesWithLockTime
	Info = append(Info, payload.VotesWithLockTime{
		Candidate: candidate1,
		Votes:     20,
		LockTime:  11150,
	})

	detailedDPoSV2Votes[*programHash1] = make(map[common.Uint256]payload.DetailedVoteInfo)
	voteInfo := &payload.DetailedVoteInfo{
		StakeProgramHash: *programHash1,
		TransactionHash:  *txhash1,
		BlockHeight:      1,
		PayloadVersion:   2,
		VoteType:         1,
		Info:             Info,
	}
	refKey := voteInfo.ReferKey()
	detailedDPoSV2Votes[*programHash1][refKey] = *voteInfo
	buf := new(bytes.Buffer)
	SerializeDetailVoteInfoMap(detailedDPoSV2Votes, buf)
	detailedDPoSV2Votes2, _ := DeserializeDetailVoteInfoMap(buf)
	assert.Equal(t, detailedDPoSV2Votes, detailedDPoSV2Votes2)
	fmt.Println(detailedDPoSV2Votes2)
}

func TestGenerateStakeAddress(t *testing.T) {
	code := getCode("03878cbe6abdafc702befd90e2329c4f37e7cb166410f0ecb70488c74c85b81d66")
	ct, _ := contract.CreateStakeContractByCode(code)
	programHash := ct.ToProgramHash()
	stakeAddress, _ := programHash.ToAddress()
	fmt.Println(stakeAddress)
}

func getCode(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}

func TestBreakOut(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println("i ", i)
	out:
		for j := 11; j < 20; j++ {
			for k := 21; k < 30; k++ {
				fmt.Println("j", j, "k ", k)
				break out
			}
		}
	}

}
