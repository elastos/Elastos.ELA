// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"github.com/elastos/Elastos.ELA/crypto"
)

func (s *txValidatorSpecialTxTestSuite) TestCheckSidechainIllegalEvidence() {
	illegalData := &payload.SidechainIllegalData{
		IllegalType: payload.IllegalBlock, // set illegal type
	}
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"invalid type")

	illegalData.IllegalType = payload.SidechainIllegalProposal
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"the encodeData cann't be nil")

	illegalData.IllegalSigner = randomBytes(33)
	//s.EqualError(CheckSidechainIllegalEvidence(illegalData),
	//	"the encodeData format is error")

	_, pk, _ := crypto.GenerateKeyPair()
	illegalData.IllegalSigner, _ = pk.EncodePoint(true)
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"illegal signer is not one of current arbitrators")

	illegalData.IllegalSigner = s.arbitrators.CurrentArbitrators[0].GetNodePublicKey()
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"[Uint168FromAddress] error, len != 34")

	illegalData.GenesisBlockAddress = "8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta"
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"insufficient signs count")

	for i := 0; i < 4; i++ {
		s, _ := crypto.Sign(s.arbitrators.CurrentArbitrators[0].GetNodePublicKey(),
			illegalData.Data(payload.SidechainIllegalDataVersion))
		illegalData.Signs = append(illegalData.Signs, s)
	}
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"evidence order error")

	// same data hash will emit order error
	evidence := &payload.SidechainIllegalEvidence{}
	cmpEvidence := &payload.SidechainIllegalEvidence{}
	evidence.DataHash = *randomUint256()
	cmpEvidence.DataHash = evidence.DataHash
	illegalData.Evidence = *evidence
	illegalData.CompareEvidence = *cmpEvidence
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"evidence order error")

	cmpEvidence.DataHash = *randomUint256()
	asc := evidence.DataHash.Compare(cmpEvidence.DataHash) < 0
	if asc {
		illegalData.Evidence = *cmpEvidence
		illegalData.CompareEvidence = *evidence
	} else {
		illegalData.Evidence = *evidence
		illegalData.CompareEvidence = *cmpEvidence
	}
	s.EqualError(blockchain.CheckSidechainIllegalEvidence(illegalData),
		"evidence order error")

	if asc {
		illegalData.Evidence = *evidence
		illegalData.CompareEvidence = *cmpEvidence
	} else {
		illegalData.Evidence = *cmpEvidence
		illegalData.CompareEvidence = *evidence
	}
	s.NoError(blockchain.CheckSidechainIllegalEvidence(illegalData))
}
