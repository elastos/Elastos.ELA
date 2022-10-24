package core

import (
	"time"

	common2 "github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

var (
	// ELAAssetID represents the asset ID of ELA coin.
	ELAAssetID = "b037db964a231458d2d6ffd5ea18944c4f90e63d547c5d3b9874df66a4ead0a3"

	// attrNonce represents the nonce attribute used in the genesis coinbase transaction.
	attrNonce = common.NewAttribute(common.Nonce, []byte{77, 101, 130, 33, 7, 252, 253, 82})

	// ELAPrecision represents the precision of ELA coin.
	ELAPrecision = byte(0x08)

	// genesisTime indicates the time when ELA genesis block created.
	genesisTime, _ = time.Parse(time.RFC3339, "2017-12-22T10:00:00Z")

	// zeroHash represents a hash with all '0' value.
	zeroHash = common2.Uint256{}
)

// GenesisBlock creates a genesis block by the specified foundation address.
// The genesis block goes different because the foundation address in each network is different.
func GenesisBlock(foundationAddr string) *types.Block {
	foundation, _ := common2.Uint168FromAddress(foundationAddr)
	// elaAsset is the transaction that create and register the ELA coin.
	elaAsset := functions.CreateTransaction(
		0,
		common.RegisterAsset,
		0,
		&payload.RegisterAsset{
			Asset: payload.Asset{
				Name:      "ELA",
				Precision: ELAPrecision,
				AssetType: 0x00,
			},
			Amount:     0 * 100000000,
			Controller: common2.Uint168{},
		},
		[]*common.Attribute{},
		[]*common.Input{},
		[]*common.Output{},
		0,
		[]*program.Program{},
	)

	assetID, _ := common2.Uint256FromHexString(ELAAssetID)
	coinBase := functions.CreateTransaction(
		0,
		common.CoinBase,
		payload.CoinBaseVersion,
		&payload.CoinBase{},
		[]*common.Attribute{&attrNonce},
		[]*common.Input{
			{
				Previous: common.OutPoint{
					TxID:  zeroHash,
					Index: 0x0000,
				},
				Sequence: 0x00000000,
			},
		},
		[]*common.Output{
			{
				AssetID:     *assetID,
				Value:       3300 * 10000 * 100000000,
				ProgramHash: *foundation,
			},
		},
		0,
		[]*program.Program{},
	)

	merkleRoot, _ := crypto.ComputeRoot([]common2.Uint256{coinBase.Hash(), *assetID})

	return &types.Block{
		Header: common.Header{
			Version:    0,
			Previous:   zeroHash,
			MerkleRoot: merkleRoot,
			Timestamp:  uint32(genesisTime.Unix()),
			Bits:       0x1d03ffff,
			Nonce:      2083236893,
			Height:     0,
		},
		Transactions: []interfaces.Transaction{coinBase, elaAsset},
	}
}
