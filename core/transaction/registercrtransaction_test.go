package transaction

import (
	"bytes"
	"encoding/hex"
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	crstate "github.com/elastos/Elastos.ELA/cr/state"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorTestSuite) TestCheckRegisterCRTransaction() {
	config.DefaultParams = *config.GetDefaultParams()

	// Generate a register CR transaction
	publicKeyStr1 := "03c77af162438d4b7140f8544ad6523b9734cca9c7a62476d54ed5d1bddc7a39c3"
	privateKeyStr1 := "7638c2a799d93185279a4a6ae84a5b76bd89e41fa9f465d9ae9b2120533983a1"
	publicKeyStr2 := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	privateKeyStr2 := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"
	publicKeyStr3 := "024010e8ac9b2175837dac34917bdaf3eb0522cff8c40fc58419d119589cae1433"
	privateKeyStr3 := "e19737ffeb452fc7ed9dc0e70928591c88ad669fd1701210dcd8732e0946829b"
	nickName1 := randomString()

	hash1, _ := getDepositAddress(publicKeyStr1)
	hash2, _ := getDepositAddress(publicKeyStr2)

	txn := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoVersion, &common.Uint168{})

	code1 := getCodeByPubKeyStr(publicKeyStr1)
	code2 := getCodeByPubKeyStr(publicKeyStr2)
	codeStr1 := common.BytesToHexString(code1)

	cid1 := getCID(code1)
	cid2 := getCID(code2)

	votingHeight := config.DefaultParams.CRConfiguration.CRVotingStartHeight
	registerCRByDIDHeight := config.DefaultParams.CRConfiguration.RegisterCRByDIDHeight

	// All ok
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ := txn.SpecialContextCheck()
	s.NoError(err)

	// Give an invalid NickName length 0 in payload
	nickName := txn.Payload().(*payload.CRInfo).NickName
	txn.Payload().(*payload.CRInfo).NickName = ""
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid NickName length more than 100 in payload
	txn.Payload().(*payload.CRInfo).NickName = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field NickName has invalid string length")

	// Give an invalid url length more than 100 in payload
	url := txn.Payload().(*payload.CRInfo).Url
	txn.Payload().(*payload.CRInfo).NickName = nickName
	txn.Payload().(*payload.CRInfo).Url = "012345678901234567890123456789012345678901234567890" +
		"12345678901234567890123456789012345678901234567890123456789"
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:field Url has invalid string length")

	// Not in vote Period lower
	txn.Payload().(*payload.CRInfo).Url = url
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: config.DefaultParams.CRConfiguration.CRVotingStartHeight - 1,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	// Not in vote Period upper c.params.CRCommitteeStartHeight
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

	// Nickname already in use
	s.Chain.GetCRCommittee().GetState().Nicknames[nickName1] = struct{}{}
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:nick name "+nickName1+" already inuse")

	delete(s.Chain.GetCRCommittee().GetState().Nicknames, nickName1)
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: 0,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:should create tx during voting period")

	delete(s.Chain.GetCRCommittee().GetState().CodeCIDMap, codeStr1)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.NoError(err)

	// CID already exist
	s.Chain.GetCRCommittee().GetState().CodeCIDMap[codeStr1] = *cid1
	s.Chain.GetCRCommittee().GetState().Candidates[*cid1] = &crstate.Candidate{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:cid "+cid1.String()+" already exist")
	delete(s.Chain.GetCRCommittee().GetState().Candidates, *cid1)

	// Give an invalid code in payload
	txn.Payload().(*payload.CRInfo).Code = []byte{}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:code is nil")

	// Give an invalid CID in payload
	txn.Payload().(*payload.CRInfo).Code = code1
	txn.Payload().(*payload.CRInfo).CID = common.Uint168{1, 2, 3}
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Give a mismatching code and CID in payload
	txn.Payload().(*payload.CRInfo).CID = *cid2
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid cid address")

	// Invalidates the signature in payload
	txn.Payload().(*payload.CRInfo).CID = *cid1
	signatature := txn.Payload().(*payload.CRInfo).Signature
	txn.Payload().(*payload.CRInfo).Signature = randomSignature()
	err, _ = txn.SpecialContextCheck()
	txn.Payload().(*payload.CRInfo).Signature = signatature
	s.EqualError(err, "transaction validate error: payload content invalid:[Validation], Verify failed.")

	// Give a mismatching deposit address
	outPuts := txn.Outputs()
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash2,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	err, _ = txn.SpecialContextCheck()
	txn.SetOutputs(outPuts)
	s.EqualError(err, "transaction validate error: payload content invalid:deposit address does not match the code in payload")

	// Give a insufficient deposit coin
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       4000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash1,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	err, _ = txn.SpecialContextCheck()
	txn.SetOutputs(outPuts)
	s.EqualError(err, "transaction validate error: payload content invalid:CR deposit amount is insufficient")

	// Multi deposit addresses
	txn.SetOutputs([]*common2.Output{
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *hash1,
			Payload:     new(outputpayload.DefaultOutput),
		},
		{
			AssetID:     common.Uint256{},
			Value:       5000 * 100000000,
			OutputLock:  0,
			ProgramHash: *hash1,
			Payload:     new(outputpayload.DefaultOutput),
		}})
	err, _ = txn.SpecialContextCheck()
	txn.SetOutputs(outPuts)
	s.EqualError(err, "transaction validate error: payload content invalid:there must be only one deposit address in outputs")

	// Check correct register CR transaction with multi sign code.
	txn = s.getMultiSigRegisterCRTx(
		[]string{publicKeyStr1, publicKeyStr2, publicKeyStr3},
		[]string{privateKeyStr1, privateKeyStr2, privateKeyStr3}, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR not support multi sign code")

	txn = s.getMultiSigRegisterCRTx(
		[]string{publicKeyStr1, publicKeyStr2, publicKeyStr3},
		[]string{privateKeyStr1, privateKeyStr2}, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR not support multi sign code")

	txn = s.getMultiSigRegisterCRTx(
		[]string{publicKeyStr1, publicKeyStr2, publicKeyStr3},
		[]string{privateKeyStr1}, nickName1)
	txn = CreateTransactionByType(txn, s.Chain)
	txn.SetParameters(&TransactionParameters{
		Transaction: txn,
		BlockHeight: votingHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:CR not support multi sign code")

	//check register cr with CRInfoDIDVersion
	txn2 := s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoDIDVersion, &common.Uint168{1, 2, 3})
	txn2 = CreateTransactionByType(txn2, s.Chain)
	txn2.SetParameters(&TransactionParameters{
		Transaction: txn2,
		BlockHeight: registerCRByDIDHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid did address")
	did2, _ := blockchain.GetDIDFromCode(code2)
	txn2 = s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoDIDVersion, did2)
	txn2 = CreateTransactionByType(txn2, s.Chain)
	txn2.SetParameters(&TransactionParameters{
		Transaction: txn2,
		BlockHeight: registerCRByDIDHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()
	s.EqualError(err, "transaction validate error: payload content invalid:invalid did address")

	did1, _ := blockchain.GetDIDFromCode(code1)
	txn2 = s.getRegisterCRTx(publicKeyStr1, privateKeyStr1, nickName1,
		payload.CRInfoDIDVersion, did1)
	txn2 = CreateTransactionByType(txn2, s.Chain)
	txn2.SetParameters(&TransactionParameters{
		Transaction: txn2,
		BlockHeight: registerCRByDIDHeight,
		TimeStamp:   s.Chain.BestChain.Timestamp,
		Config:      s.Chain.GetParams(),
		BlockChain:  s.Chain,
	})
	err, _ = txn2.SpecialContextCheck()
	s.NoError(err)
}

func (s *txValidatorTestSuite) getRegisterCRTx(publicKeyStr, privateKeyStr,
	nickName string, payloadVersion byte, did *common.Uint168) interfaces.Transaction {

	publicKeyStr1 := publicKeyStr
	privateKeyStr1 := privateKeyStr
	publicKey1, _ := common.HexStringToBytes(publicKeyStr1)
	privateKey1, _ := common.HexStringToBytes(privateKeyStr1)

	code1 := getCodeByPubKeyStr(publicKeyStr1)
	ct1, _ := contract.CreateCRIDContractByCode(code1)
	cid1 := ct1.ToProgramHash()

	hash1, _ := contract.PublicKeyToDepositProgramHash(publicKey1)

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.RegisterCR,
		payloadVersion,
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
		DID:      *did,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}
	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payloadVersion)
	rcSig1, _ := crypto.Sign(privateKey1, signBuf.Bytes())
	crInfoPayload.Signature = rcSig1
	txn.SetPayload(crInfoPayload)

	txn.SetPrograms([]*program.Program{{
		Code:      getCodeByPubKeyStr(publicKeyStr1),
		Parameter: nil,
	}})

	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *hash1,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}

func (s *txValidatorTestSuite) getMultiSigRegisterCRTx(
	publicKeyStrs, privateKeyStrs []string, nickName string) interfaces.Transaction {

	var publicKeys []*crypto.PublicKey
	for _, publicKeyStr := range publicKeyStrs {
		publicKeyBytes, _ := hex.DecodeString(publicKeyStr)
		publicKey, _ := crypto.DecodePoint(publicKeyBytes)
		publicKeys = append(publicKeys, publicKey)
	}

	multiCode, _ := contract.CreateMultiSigRedeemScript(len(publicKeys)*2/3, publicKeys)

	ctDID, _ := contract.CreateCRIDContractByCode(multiCode)
	cid := ctDID.ToProgramHash()

	ctDeposit, _ := contract.CreateDepositContractByCode(multiCode)
	deposit := ctDeposit.ToProgramHash()

	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.RegisterCR,
		0,
		nil,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)
	crInfoPayload := &payload.CRInfo{
		Code:     multiCode,
		CID:      *cid,
		NickName: nickName,
		Url:      "http://www.elastos_test.com",
		Location: 1,
	}

	signBuf := new(bytes.Buffer)
	crInfoPayload.SerializeUnsigned(signBuf, payload.CRInfoVersion)
	for _, privateKeyStr := range privateKeyStrs {
		privateKeyBytes, _ := hex.DecodeString(privateKeyStr)
		sig, _ := crypto.Sign(privateKeyBytes, signBuf.Bytes())
		crInfoPayload.Signature = append(crInfoPayload.Signature, byte(len(sig)))
		crInfoPayload.Signature = append(crInfoPayload.Signature, sig...)
	}

	txn.SetPayload(crInfoPayload)
	txn.SetPrograms([]*program.Program{{
		Code:      multiCode,
		Parameter: nil,
	}})
	txn.SetOutputs([]*common2.Output{{
		AssetID:     common.Uint256{},
		Value:       5000 * 100000000,
		OutputLock:  0,
		ProgramHash: *deposit,
		Type:        0,
		Payload:     new(outputpayload.DefaultOutput),
	}})
	return txn
}
