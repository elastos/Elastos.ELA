package transaction

import (
	"bytes"
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/checkpoint"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	"github.com/elastos/Elastos.ELA/core/types"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
	"path/filepath"
)

func (s *txValidatorTestSuite) TestCheckUpdateCRTransaction() {

	// Generate a UpdateCR CR transaction
	publicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	privateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"

	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"

	nickName1 := "nickname 1"
	nickName2 := "nickname 2"
	nickName3 := "nickname 3"

	votingHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight
	//
	//registe an cr to update
	registerCRTxn1 := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1,
		nickName1, payload.CRInfoVersion, &common.Uint168{})
	registerCRTxn2 := s.getRegisterCRTx(publicKeyStr2, privateKeyStr2,
		nickName2, payload.CRInfoDIDVersion, &common.Uint168{})

	s.CurrentHeight = s.Chain.GetParams().CRConfiguration.CRVotingStartHeight + 1
	ckpManager := checkpoint.NewManager(&config.DefaultParams)
	ckpManager.SetDataPath(filepath.Join(config.DefaultParams.DataDir, "checkpoints"))
	s.Chain.SetCRCommittee(crstate.NewCommittee(s.Chain.GetParams(), ckpManager))
	s.Chain.GetCRCommittee().RegisterFuncitons(&crstate.CommitteeFuncsConfig{
		GetTxReference:                   s.Chain.UTXOCache.GetTxReference,
		GetUTXO:                          s.Chain.GetDB().GetFFLDB().GetUTXO,
		GetHeight:                        func() uint32 { return s.CurrentHeight },
		CreateCRAppropriationTransaction: s.Chain.CreateCRCAppropriationTransaction,
	})
	block := &types.Block{
		Transactions: []interfaces.Transaction{
			registerCRTxn1,
			registerCRTxn2,
		},
		Header: common2.Header{Height: s.CurrentHeight},
	}
	s.Chain.GetCRCommittee().ProcessBlock(block, nil)

	//ok nothing wrong
	hash2, err := getDepositAddress(publicKeyStr2)
	txn := s.getUpdateCRTx(publicKeyStr1, privateKeyStr1, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid NickName length 0 in payload
	nickName := txn.Payload().(*payload.CRInfo).NickName
	txn.Payload().(*payload.CRInfo).NickName = ""
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).NickName = nickName
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid NickName length more than 100 in payload
	txn.Payload().(*payload.CRInfo).NickName = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).NickName = nickName
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid url length more than 100 in payload
	url := txn.Payload().(*payload.CRInfo).Url
	txn.Payload().(*payload.CRInfo).Url = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Url = url
	s.EqualError(err, "transaction validate error: payload content invalid:field Url has invalid string length")

	// Give an invalid code in payload
	code := txn.Payload().(*payload.CRInfo).Code
	txn.Payload().(*payload.CRInfo).Code = []byte{1, 2, 3, 4, 5}
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Code = code
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Give an invalid CID in payload
	cid := txn.Payload().(*payload.CRInfo).CID
	txn.Payload().(*payload.CRInfo).CID = common.Uint168{1, 2, 3}
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).CID = cid
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Give a mismatching code and CID in payload
	txn.Payload().(*payload.CRInfo).CID = *hash2
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).CID = cid
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Invalidates the signature in payload
	signatur := txn.Payload().(*payload.CRInfo).Signature
	txn.Payload().(*payload.CRInfo).Signature = randomSignature()
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Signature = signatur
	s.EqualError(err, "transaction validate error: payload content invalid:[Validation], Verify failed.")

	//not in vote Period lower
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRConfiguration.CRVotingStartHeight - 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	// set RegisterCRByDIDHeight after CRCommitteeStartHeight
	s.Chain.GetParams().CRConfiguration.RegisterCRByDIDHeight = config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 10

	//not in vote Period lower upper c.params.CRCommitteeStartHeight
	s.Chain.GetCRCommittee().InElectionPeriod = true
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRConfiguration.CRCommitteeStartHeight + 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	//updating unknown CR
	txn3 := s.getUpdateCRTx(publicKeyStr3, privateKeyStr3, nickName3)
	txn3 = CreateTransactionByType(txn3, s.Chain)
	txn3.SetParameters(&TransactionParameters{
		Transaction: txn3,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn3.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:updating unknown CR")

	//nick name already exist
	txn1Copy := s.getUpdateCRTx(publicKeyStr1, privateKeyStr1, nickName2)
	txn1Copy = CreateTransactionByType(txn1Copy, s.Chain)
	txn1Copy.SetParameters(&TransactionParameters{
		Transaction: txn1Copy,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn1Copy.SpecialContextCheck()
	str := fmt.Sprintf("transaction validate error: payload content invalid:nick name %s already exist", nickName2)
	s.EqualError(err, str)

}

func (s *txValidatorTestSuite) getUpdateCRTx(publicKeyStr, privateKeyStr, nickName string) interfaces.Transaction {

	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)
	code1 := getCodeByPubKeyStr(publicKeyStr1)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	cid1 := ct1.ToProgramHash()

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.UpdateCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	crInfoPayload := &payload.CRInfo{
		Code:     code1,
		CID:      *cid1,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	err := crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	s.NoError(err)
	rcSig1, err := crypto.Sign(privateKey1, signBuf.Bytes())
	s.NoError(err)
	crInfoPayload.Signature = rcSig1
	txn.SetPayload(crInfoPayload)

	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})
	return txn
}
