package blocks

import (
	"math"
	"testing"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types"
	"github.com/elastos/Elastos.ELA/version/verconf"
	"github.com/stretchr/testify/suite"
)

type blockVersionV0TestSuite struct {
	suite.Suite

	Version BlockVersion
	Cfg     *verconf.Config
}

func (s *blockVersionV0TestSuite) SetupTest() {
	config.Parameters = config.ConfigParams{Configuration: &config.Template}

	s.Cfg = &verconf.Config{
		ChainParams: &config.MainNetParams,
	}
	s.Version = NewBlockV0(s.Cfg)
}

func (s *blockVersionV0TestSuite) TestBlockVersionMain_GetNormalArbitersDesc() {
	originLedger := blockchain.DefaultLedger

	arbitrators := make([][]byte, 0)
	for _, v := range config.MainNetParams.OriginArbiters {
		a, _ := common.HexStringToBytes(v)
		arbitrators = append(arbitrators, a)
	}

	producers, err := s.Version.GetNormalArbitersDesc(5, cfg.Chain.GetState().GetInterfaceProducers())
	s.NoError(err)
	for i := range producers {
		s.Equal(arbitrators[i], producers[i])
	}

	blockchain.DefaultLedger = originLedger
}

func (s *blockVersionV0TestSuite) TestBlockVersionMain_AssignCoinbaseTxRewards() {
	originLedger := blockchain.DefaultLedger
	blockchain.DefaultLedger = &blockchain.Ledger{
		Blockchain: &blockchain.BlockChain{},
	}

	//reward can be exactly division

	rewardInCoinbase := common.Fixed64(1000)
	foundationReward := common.Fixed64(float64(rewardInCoinbase) * 0.3)
	minerReward := common.Fixed64(float64(rewardInCoinbase) * 0.35)
	dposTotalReward := rewardInCoinbase - foundationReward - minerReward

	tx := &types.Transaction{
		Version: types.TransactionVersion(s.Version.GetVersion()),
		TxType:  types.CoinBase,
	}
	tx.Outputs = []*types.Output{
		{ProgramHash: blockchain.FoundationAddress, Value: 0},
		{ProgramHash: common.Uint168{}, Value: 0},
	}
	block := &types.Block{
		Transactions: []*types.Transaction{
			tx,
		},
	}

	s.NoError(s.Version.AssignCoinbaseTxRewards(block, rewardInCoinbase))

	s.Equal(foundationReward, tx.Outputs[0].Value)
	s.Equal(minerReward, tx.Outputs[1].Value)
	s.Equal(dposTotalReward, tx.Outputs[1].Value)

	//reward can not be exactly division

	rewardInCoinbase = common.Fixed64(999)
	foundationReward = common.Fixed64(math.Ceil(float64(rewardInCoinbase) * 0.3))
	foundationRewardNormal := common.Fixed64(float64(rewardInCoinbase) * 0.3)
	minerReward = common.Fixed64(float64(rewardInCoinbase) * 0.35)
	dposTotalReward = rewardInCoinbase - foundationRewardNormal - minerReward

	tx = &types.Transaction{
		Version: types.TransactionVersion(s.Version.GetVersion()),
		TxType:  types.CoinBase,
	}
	tx.Outputs = []*types.Output{
		{ProgramHash: blockchain.FoundationAddress, Value: 0},
		{ProgramHash: common.Uint168{}, Value: 0},
	}
	block = &types.Block{
		Transactions: []*types.Transaction{
			tx,
		},
	}

	s.NoError(s.Version.AssignCoinbaseTxRewards(block, rewardInCoinbase))

	s.Equal(foundationRewardNormal, tx.Outputs[0].Value)
	s.NotEqual(foundationReward, tx.Outputs[0].Value)
	s.Equal(minerReward, tx.Outputs[1].Value)
	s.Equal(dposTotalReward, tx.Outputs[2].Value)

	blockchain.DefaultLedger = originLedger
}

func (s *blockVersionV0TestSuite) TestBlockVersionMain_GetNextOnDutyArbiter() {

	// fixme chain store removed, fix me later
	//arbitrators := make([][]byte, 0)
	//for _, v := range config.MainNetParams.OriginArbiters {
	//	a, _ := common.HexStringToBytes(v)
	//	arbitrators = append(arbitrators, a)
	//}
	//
	//originLedger := blockchain.DefaultLedger
	//chainMock := &blockchain.ChainStoreMock{
	//	BlockHeight: 0,
	//}
	//blockchain.DefaultLedger = &blockchain.Ledger{
	//	Store: chainMock,
	//}
	//
	//var currentArbiter []byte
	//
	//chainMock.BlockHeight = 0
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(1, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(2, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(3, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(4, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(5, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//
	//chainMock.BlockHeight = 0
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//
	//chainMock.BlockHeight = 1
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[1], currentArbiter)
	//
	//chainMock.BlockHeight = 2
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[2], currentArbiter)
	//
	//chainMock.BlockHeight = 3
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[3], currentArbiter)
	//
	//chainMock.BlockHeight = 4
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[4], currentArbiter)
	//
	//chainMock.BlockHeight = 5
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 0)
	//s.Equal(arbitrators[0], currentArbiter)
	//
	//chainMock.BlockHeight = 0
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 1)
	//s.Equal(arbitrators[1], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 2)
	//s.Equal(arbitrators[2], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 3)
	//s.Equal(arbitrators[3], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 4)
	//s.Equal(arbitrators[4], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 5)
	//s.Equal(arbitrators[0], currentArbiter)
	//currentArbiter = s.Version.GetNextOnDutyArbiter(0, 6)
	//s.Equal(arbitrators[1], currentArbiter)
	//
	//blockchain.DefaultLedger = originLedger
}

func TestBlockVersionV0Suit(t *testing.T) {
	suite.Run(t, new(blockVersionV0TestSuite))
}
