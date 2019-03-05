package version

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types"
)

// Ensure HeightVersionsMock implements the HeightVersions interface.
var _ HeightVersions = (*HeightVersionsMock)(nil)

type HeightVersionsMock struct {
	Producers           [][]byte
	ShouldConfirm       bool
	CurrentArbiter   []byte
	DefaultTxVersion    byte
	DefaultBlockVersion uint32
}

func NewMock() *HeightVersionsMock {
	const arbitratorStr = "8a6cb4b5ff1a4f8368c6513a536c663381e3fdeff738e9b437bd8fce3fb30b62"
	arbitrator, _ := common.HexStringToBytes(arbitratorStr)

	mockObj := &HeightVersionsMock{
		Producers:           make([][]byte, 0),
		ShouldConfirm:       true,
		CurrentArbiter:   arbitrator,
		DefaultTxVersion:    1,
		DefaultBlockVersion: 1,
	}

	return mockObj
}

func (b *HeightVersionsMock) GetCandidatesDesc(blockHeight uint32, startIndex uint32, producers []Producer) ([][]byte, error) {
	return nil, nil
}

func (b *HeightVersionsMock) GetNormalArbitersDesc(blockHeight uint32, arbitratorsCount uint32, arbiters []Producer) ([][]byte, error) {
	return nil, nil
}

func (b *HeightVersionsMock) GetDefaultTxVersion(blockHeight uint32) byte {
	return b.DefaultTxVersion
}

func (b *HeightVersionsMock) GetDefaultBlockVersion(blockHeight uint32) uint32 {
	return b.DefaultBlockVersion
}

func (b *HeightVersionsMock) CheckOutputProgramHash(blockHeight uint32, tx *types.Transaction, programHash common.Uint168) error {
	return nil
}

func (b *HeightVersionsMock) CheckCoinbaseMinerReward(blockHeight uint32, tx *types.Transaction, totalReward common.Fixed64) error {
	return nil
}

func (b *HeightVersionsMock) CheckCoinbaseArbitersReward(blockHeight uint32, coinbase *types.Transaction, rewardInCoinbase common.Fixed64) error {
	return nil
}

func (b *HeightVersionsMock) AddBlock(block *types.Block) (bool, bool, error) {
	return true, false, nil
}

func (b *HeightVersionsMock) AddDposBlock(block *types.DposBlock) (bool, bool, error) {
	return true, false, nil
}

func (b *HeightVersionsMock) AssignCoinbaseTxRewards(block *types.Block, totalReward common.Fixed64) error {
	return nil
}

func (b *HeightVersionsMock) CheckConfirmedBlockOnFork(block *types.Block) error {
	return nil
}

func (b *HeightVersionsMock) GetNextOnDutyArbiter(blockHeight, dutyChangedCount, offset uint32) []byte {
	return b.CurrentArbiter
}
