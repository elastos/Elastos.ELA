// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

func (s *txValidatorSpecialTxTestSuite) TestCheckInactiveArbitrators() {
	p := &payload.InactiveArbitrators{
		Sponsor: randomPublicKey(),
	}
	tx := functions.CreateTransaction(
		0,
		common2.RegisterCR,
		payload.CRInfoDIDVersion,
		p,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{
			{
				Code:      randomPublicKey(),
				Parameter: randomSignature(),
			},
		},
	)

	s.arbitrators.ActiveProducer = s.arbitrators.CurrentArbitrators

	s.EqualError(blockchain.CheckInactiveArbitrators(tx),
		"sponsor is not belong to arbitrators")

	// correct sponsor
	ar, _ := state.NewOriginArbiter(p.Sponsor)
	s.arbitrators.CRCArbitrators = []state.ArbiterMember{ar}
	for i := 0; i < 3; i++ { // add more than InactiveEliminateCount arbiters
		p.Arbitrators = append(p.Arbitrators,
			s.arbitrators.CurrentArbitrators[i].GetNodePublicKey())
	}

	// correct number of Arbitrators
	p.Arbitrators = make([][]byte, 0)
	p.Arbitrators = append(p.Arbitrators, randomPublicKey())
	s.EqualError(blockchain.CheckInactiveArbitrators(tx),
		"inactive arbitrator is not belong to arbitrators")

	// correct "Arbitrators" to be current arbitrators
	p.Arbitrators = make([][]byte, 0)
	for i := 4; i < 5; i++ {
		p.Arbitrators = append(p.Arbitrators,
			s.arbitrators.CurrentArbitrators[i].GetNodePublicKey())
	}
	s.EqualError(blockchain.CheckInactiveArbitrators(tx),
		"invalid multi sign script code")

	// let "Arbitrators" has CRC arbitrators
	ar, _ = state.NewOriginArbiter(p.Sponsor)
	s.arbitrators.CRCArbitrators = []state.ArbiterMember{
		ar,
		s.arbitrators.CurrentArbitrators[4],
	}
	s.EqualError(blockchain.CheckInactiveArbitrators(tx),
		"inactive arbiters should not include CRC")

	// set invalid redeem script
	s.arbitrators.CRCArbitrators = []state.ArbiterMember{}
	for i := 0; i < 5; i++ {
		_, pk, _ := crypto.GenerateKeyPair()
		pkBuf, _ := pk.EncodePoint(true)
		ar, _ = state.NewOriginArbiter(pkBuf)
		s.arbitrators.CRCArbitrators = append(s.arbitrators.CRCArbitrators, ar)
	}
	s.arbitrators.CRCArbitratorsMap = map[string]*state.Producer{}
	for _, v := range s.arbitrators.CRCArbitrators {
		s.arbitrators.CRCArbitratorsMap[common.BytesToHexString(
			v.GetNodePublicKey())] = nil
	}
	p.Sponsor = s.arbitrators.CRCArbitrators[0].GetNodePublicKey()
	var arbitrators []state.ArbiterMember
	for i := 0; i < 4; i++ {
		arbitrators = append(arbitrators, s.arbitrators.CurrentArbitrators[i])
	}
	_, pk, _ := crypto.GenerateKeyPair()
	pkBuf, _ := pk.EncodePoint(true)
	ar, _ = state.NewOriginArbiter(pkBuf)
	arbitrators = append(arbitrators, ar)
	tx.Programs()[0].Code = s.createArbitratorsRedeemScript(arbitrators)
	s.EqualError(blockchain.CheckInactiveArbitrators(tx),
		"invalid multi sign public key")

	// correct redeem script
	tx.Programs()[0].Code = s.createArbitratorsRedeemScript(
		s.arbitrators.CRCArbitrators)
	s.NoError(blockchain.CheckInactiveArbitrators(tx))
}

func (s *txValidatorSpecialTxTestSuite) createArbitratorsRedeemScript(
	arbitrators []state.ArbiterMember) []byte {

	var pks []*crypto.PublicKey
	for _, v := range arbitrators {
		pk, err := crypto.DecodePoint(v.GetNodePublicKey())
		if err != nil {
			return nil
		}
		pks = append(pks, pk)
	}

	arbitratorsCount := len(arbitrators)
	minSignCount := arbitratorsCount * 2 / 3
	result, _ := contract.CreateMultiSigRedeemScript(minSignCount+1, pks)
	return result
}
