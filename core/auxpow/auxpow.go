package auxpow

import (
	. "DNA_POW/common"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strings"
)

var (
	pchMergedMiningHeader = []byte{0xfa, 0xbe, 'm', 'm'}
)

type AuxPow struct {
	AuxMerkleBranch   []Uint256
	AuxMerkleIndex    int
	ParCoinbaseTx     BtcTx
	ParCoinBaseMerkle []Uint256
	ParMerkleIndex    int
	ParBlockHeader    BtcBlockHeader
}

func NewAuxPow(AuxMerkleBranch []Uint256, AuxMerkleIndex int,
	ParCoinbaseTx BtcTx, ParCoinBaseMerkle []Uint256,
	ParMerkleIndex int, ParBlockHeader BtcBlockHeader) *AuxPow {

	return &AuxPow{
		AuxMerkleBranch:   AuxMerkleBranch,
		AuxMerkleIndex:    AuxMerkleIndex,
		ParCoinbaseTx:     ParCoinbaseTx,
		ParCoinBaseMerkle: ParCoinBaseMerkle,
		ParMerkleIndex:    ParMerkleIndex,
		ParBlockHeader:    ParBlockHeader,
	}
}

func (ap *AuxPow) Check(hashAuxBlock Uint256, chainId int) bool {
	if CheckMerkleBranch(ap.ParCoinbaseTx.Hash(), ap.ParCoinBaseMerkle, ap.ParMerkleIndex) != ap.ParBlockHeader.MerkleRoot {
		return false
	}

	auxRootHash := CheckMerkleBranch(hashAuxBlock, ap.AuxMerkleBranch, ap.AuxMerkleIndex)

	script := ap.ParCoinbaseTx.TxIn[0].SignatureScript
	scriptStr := hex.EncodeToString(script)
	//fixme reverse
	auxRootHashStr := hex.EncodeToString(auxRootHash.ToArray())
	pchMergedMiningHeaderStr := hex.EncodeToString(pchMergedMiningHeader)

	headerIndex := strings.Index(scriptStr, pchMergedMiningHeaderStr)
	rootHashIndex := strings.Index(scriptStr, auxRootHashStr)

	if (headerIndex == -1) || (rootHashIndex == -1) {
		return false
	}

	if strings.Index(scriptStr[headerIndex+2:], pchMergedMiningHeaderStr) != -1 {
		return false
	}

	if headerIndex+len(pchMergedMiningHeaderStr) != rootHashIndex {
		return false
	}

	rootHashIndex += len(auxRootHashStr)
	if len(scriptStr)-rootHashIndex < 8 {
		return false
	}

	size := binary.LittleEndian.Uint32(script[rootHashIndex/2 : rootHashIndex/2+4])
	merkleHeight := len(ap.AuxMerkleBranch)
	if size != uint32(1<<uint32(merkleHeight)) {
		return false
	}

	nonce := binary.LittleEndian.Uint32(script[rootHashIndex/2+4 : rootHashIndex/2+8])
	if ap.AuxMerkleIndex != GetExpectedIndex(nonce, chainId, merkleHeight) {
		return false
	}

	return true
}

func CheckMerkleBranch(hash Uint256, merkleBranch []Uint256, index int) Uint256 {
	if index == -1 {
		return Uint256{}
	}
	for _, it := range merkleBranch {
		if (index & 1) == 1 {
			temp := make([]uint8, 0)
			temp = append(temp, it[:]...)
			temp = append(temp, hash[:]...)
			hash = Uint256(sha256.Sum256(temp))
		} else {
			temp := make([]uint8, 0)
			temp = append(temp, hash[:]...)
			temp = append(temp, it[:]...)
			hash = Uint256(sha256.Sum256(temp))
		}
		index >>= 1
	}
	return hash
}

func GetExpectedIndex(nonce uint32, chainId, h int) int {
	rand := nonce
	rand = rand*1103515245 + 12345
	rand += uint32(chainId)
	rand = rand*1103515245 + 12345

	return int(rand % (1 << uint32(h)))
}

func reverse(input []byte) []byte {
	if len(input) == 0 {
		return input
	}
	return append(reverse(input[1:]), input[0])
}
