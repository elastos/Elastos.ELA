package transaction

import (
	"math"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestCheckDuplicateSidechainTx() {
	hashStr1 := "8a6cb4b5ff1a4f8368c6513a536c663381e3fdeff738e9b437bd8fce3fb30b62"
	hashBytes1, _ := common.HexStringToBytes(hashStr1)
	hash1, _ := common.Uint256FromBytes(hashBytes1)
	hashStr2 := "cc62e14f5f9526b7f4ff9d34dcd0643dacb7886707c57f49ec97b95ec5c4edac"
	hashBytes2, _ := common.HexStringToBytes(hashStr2)
	hash2, _ := common.Uint256FromBytes(hashBytes2)

	// 1. Generate the ill withdraw transaction which have duplicate sidechain tx
	pd := &payload.WithdrawFromSideChain{
		BlockHeight:         100,
		GenesisBlockAddress: "eb7adb1fea0dd6185b09a43bdcd4924bb22bff7151f0b1b4e08699840ab1384b",
		SideChainTransactionHashes: []common.Uint256{
			*hash1,
			*hash2,
			*hash1, // duplicate tx hash
		},
	}

	txn := functions.CreateTransaction(
		0,
		common2.WithdrawFromSideChain,
		0,
		pd,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	// 2. Run CheckDuplicateSidechainTx
	err := blockchain.CheckDuplicateSidechainTx(txn)
	s.EqualError(err, "Duplicate sidechain tx detected in a transaction")
}

// TestSchnorrWithdrawFromSidechainSignerChecksActivateAtRestrictionHeight
// keeps legacy duplicate-index behavior before H and enables the hardening at H.
func (s *txValidatorSpecialTxTestSuite) TestSchnorrWithdrawFromSidechainSignerChecksActivateAtRestrictionHeight() {
	chainParams := *s.Chain.GetParams()
	chainParams.CrossChainUTXOFreezeHeight = 0
	chainParams.CrossChainUTXORestrictionHeight = 100
	chainParams.CRConfiguration.MemberCount = 1
	chainParams.CRConfiguration.CRClaimDPOSNodeStartHeight = 1
	chainParams.DPoSConfiguration.DPOSNodeCrossChainHeight = math.MaxUint32

	withdrawPayload := &payload.WithdrawFromSideChain{Signers: []uint8{0, 0}}
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV2,
		withdrawPayload,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	txn = CreateTransactionByType(txn, s.Chain)
	parameters := &TransactionParameters{
		Transaction: txn,
		BlockHeight: 99,
		Config:      &chainParams,
		BlockChain:  s.Chain,
	}
	txn.SetParameters(parameters)

	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	parameters.BlockHeight = 100
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:duplicate schnorr withdraw signer index")

	withdrawPayload.Signers = []uint8{0xff}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:invalid schnorr withdraw signer index")
}

// TestWithdrawFromSideChainV1ArbiterWitness verifies an Arbiter-shaped V1
// withdrawal completes sanity and contextual validation at H.
func (s *txValidatorSpecialTxTestSuite) TestWithdrawFromSideChainV1ArbiterWitness() {
	fixture := s.createArbiterCrossChainUTXOFixture()
	blockHeight := fixture.transactionHeight + 1
	chainParams := *s.Chain.GetParams()
	chainParams.CrossChainUTXOFreezeHeight = 0
	chainParams.CrossChainUTXORestrictionHeight = blockHeight
	chainParams.DPoSConfiguration.DPOSNodeCrossChainHeight = math.MaxUint32
	chainParams.CRConfiguration.CRAgreementCount = uint32(s.arbitrators.MajorityCount)

	originalCRCArbitrators := s.arbitrators.CRCArbitrators
	s.arbitrators.CRCArbitrators = s.arbitrators.CurrentArbitrators
	defer func() {
		s.arbitrators.CRCArbitrators = originalCRCArbitrators
	}()

	witness := &program.Program{Code: s.crossChainArbiterScript(
		s.arbitrators.MajorityCount, len(s.arbitrators.GetCrossChainArbiters()))}
	nonce := common2.NewAttribute(common2.Nonce, []byte("crosschain-utxo-v1"))
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.WithdrawFromSideChain,
		payload.WithdrawFromSideChainVersionV1,
		&payload.WithdrawFromSideChain{},
		[]*common2.Attribute{&nonce},
		[]*common2.Input{fixture.depositInput},
		[]*common2.Output{{
			AssetID:     core.ELAAssetID,
			Value:       fixture.reserveAmount - 100,
			ProgramHash: fixture.payerProgramHash,
			Type:        common2.OTWithdrawFromSideChain,
			Payload: &outputpayload.Withdraw{
				Version:                  outputpayload.WithdrawOutputVersion,
				GenesisBlockAddress:      fixture.bankAddress,
				SideChainTransactionHash: *randomUint256(),
				TargetData:               []byte("arbiter fixture"),
			},
		}},
		0,
		[]*program.Program{witness},
	)
	txn = CreateTransactionByType(txn, s.Chain)
	parameters := &TransactionParameters{
		Transaction: txn,
		BlockHeight: blockHeight,
		Config:      &chainParams,
		BlockChain:  s.Chain,
	}
	txn.SetParameters(parameters)
	s.signCrossChainProgram(txn, witness)

	cleanup := s.prepareArbiterCrossChainContext(&chainParams)
	defer cleanup()

	s.NoError(txn.SanityCheck(parameters))
	_, err := txn.ContextCheck(parameters)
	s.NoError(err)
}
