package config

import (
	"time"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	"github.com/elastos/Elastos.ELA/core"
)

const (
	BlockVersion uint32 = 0
)

var (
	zeroHash = common.Uint256{}

	// "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	mainNetFoundation = common.Uint168{
		0x12, 0x9e, 0x9c, 0xf1, 0xc5, 0xf3, 0x36,
		0xfc, 0xf3, 0xa6, 0xc9, 0x54, 0x44, 0x4e,
		0xd4, 0x82, 0xc5, 0xd9, 0x16, 0xe5, 0x06,
	}

	// ELA coin
	elaCoin = &core.Transaction{
		TxType:         core.RegisterAsset,
		PayloadVersion: 0,
		Payload: &core.PayloadRegisterAsset{
			Asset: core.Asset{
				Name:      "ELA",
				Precision: 0x08,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common.Uint168{},
		},
		Attributes: []*core.Attribute{},
		Inputs:     []*core.Input{},
		Outputs:    []*core.Output{},
		Programs:   []*core.Program{},
	}

	ELAAssetID = elaCoin.Hash()

	// genesisHeader
	genesisHeader = core.Header{
		Version:   BlockVersion,
		Previous:  zeroHash,
		Timestamp: uint32(time.Unix(time.Date(2017, time.December, 22, 10, 0, 0, 0, time.UTC).Unix(), 0).Unix()),
		Bits:      0x1d03ffff,
		Nonce:     2083236893,
		Height:    uint32(0),
	}

	nonceAttr = core.NewAttribute(core.Nonce, []byte{77, 101, 130, 33, 7, 252, 253, 82})
)

func genesisBlock(foundation common.Uint168) *core.Block {
	genesisCoinbase := &core.Transaction{
		TxType:         core.CoinBase,
		PayloadVersion: core.PayloadCoinBaseVersion,
		Payload:        &core.PayloadCoinBase{},
		Attributes:     []*core.Attribute{&nonceAttr},
		Inputs: []*core.Input{
			{
				Previous: core.OutPoint{
					TxID:  common.EmptyHash,
					Index: 0x0000,
				},
				Sequence: 0x00000000,
			},
		},
		Outputs: []*core.Output{
			{
				AssetID:     elaCoin.Hash(),
				Value:       3300 * 10000 * 100000000,
				ProgramHash: foundation,
			},
		},
		LockTime: 0,
		Programs: []*core.Program{},
	}

	//block
	block := &core.Block{
		Header:       genesisHeader,
		Transactions: []*core.Transaction{genesisCoinbase, elaCoin},
	}
	hashes := make([]common.Uint256, 0, len(block.Transactions))
	for _, tx := range block.Transactions {
		hashes = append(hashes, tx.Hash())
	}
	block.Header.MerkleRoot, _ = crypto.ComputeRoot(hashes)

	return block
}
