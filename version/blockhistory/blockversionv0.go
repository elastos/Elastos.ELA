package blockhistory

import (
	"errors"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/version"

	"github.com/elastos/Elastos.ELA.Utility/common"
)

var originalArbitrators = []string{
	"023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19bbc77a",
	"030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9",
	"0288e79636e41edce04d4fa95d8f62fed73a76164f8631ccc42f5425f960e4a0c7",
	"03e281f89d85b3a7de177c240c4961cb5b1f2106f09daa42d15874a38bbeae85dd",
	"0393e823c2087ed30871cbea9fa5121fa932550821e9f3b17acef0e581971efab0",
}

type BlockVersionV0 struct {
	version.BlockVersionMain
}

func (b *BlockVersionV0) GetVersion() uint32 {
	return 0
}

func (b *BlockVersionV0) GetProducersDesc() ([][]byte, error) {
	if len(originalArbitrators) == 0 {
		return nil, errors.New("arbiters not configured")
	}

	arbitersByte := make([][]byte, 0)
	for _, arbiter := range originalArbitrators {
		arbiterByte, err := common.HexStringToBytes(arbiter)
		if err != nil {
			return nil, err
		}
		arbitersByte = append(arbitersByte, arbiterByte)
	}

	return arbitersByte, nil
}

func (b *BlockVersionV0) AddBlock(block *core.Block) error {
	inMainChain, isOrphan, err := blockchain.DefaultLedger.Blockchain.AddBlock(block)
	if err != nil {
		return err
	}

	if isOrphan || !inMainChain {
		return errors.New("Append to best chain error.")
	}

	return nil
}

func (b *BlockVersionV0) AddBlockConfirm(block *core.BlockConfirm) (bool, error) {
	return false, b.AddBlock(block.Block)
}

func (b *BlockVersionV0) AssignCoinbaseTxRewards(block *core.Block, totalReward common.Fixed64) error {
	// PoW miners and DPoS are each equally allocated 35%. The remaining 30% goes to the Cyber Republic fund
	rewardCyberRepublic := common.Fixed64(float64(totalReward) * 0.3)
	rewardMergeMiner := common.Fixed64(float64(totalReward) * 0.35)
	rewardDposArbiter := common.Fixed64(totalReward) - rewardCyberRepublic - rewardMergeMiner
	block.Transactions[0].Outputs[0].Value = rewardCyberRepublic
	block.Transactions[0].Outputs[1].Value = rewardMergeMiner
	block.Transactions[0].Outputs = append(block.Transactions[0].Outputs, &core.Output{
		AssetID:     blockchain.DefaultLedger.Blockchain.AssetID,
		Value:       rewardDposArbiter,
		ProgramHash: blockchain.FoundationAddress,
	})

	return nil
}
