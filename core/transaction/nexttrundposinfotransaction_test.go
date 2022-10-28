// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract/program"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

func (s *txValidatorTestSuite) TestCheckNextTurnDPOSInfoTx() {
	crc1PubKey, _ := common.HexStringToBytes("03e435ccd6073813917c2d841a0815d21301ec3286bc1412bb5b099178c68a10b6")
	crc2PubKey, _ := common.HexStringToBytes("038a1829b4b2bee784a99bebabbfecfec53f33dadeeeff21b460f8b4fc7c2ca771")

	normalArbitratorsStr := []string{
		"023a133480176214f88848c6eaa684a54b316849df2b8570b57f3a917f19bbc77a",
		"030a26f8b4ab0ea219eb461d1e454ce5f0bd0d289a6a64ffc0743dab7bd5be0be9",
		"0288e79636e41edce04d4fa95d8f62fed73a76164f8631ccc42f5425f960e4a0c7",
		"03e281f89d85b3a7de177c240c4961cb5b1f2106f09daa42d15874a38bbeae85dd",
		"0393e823c2087ed30871cbea9fa5121fa932550821e9f3b17acef0e581971efab0",
	}
	normal1PubKey, _ := common.HexStringToBytes(normalArbitratorsStr[0])
	normal2PubKey, _ := common.HexStringToBytes(normalArbitratorsStr[1])

	crcArbiters := [][]byte{
		crc1PubKey,
		crc2PubKey,
	}
	//
	normalDPOSArbiters := [][]byte{
		normal1PubKey,
		normal2PubKey,
	}

	var nextTurnDPOSInfo payload.NextTurnDPOSInfo
	for _, v := range crcArbiters {
		nextTurnDPOSInfo.CRPublicKeys = append(nextTurnDPOSInfo.CRPublicKeys, v)
	}
	for _, v := range normalDPOSArbiters {
		nextTurnDPOSInfo.DPOSPublicKeys = append(nextTurnDPOSInfo.DPOSPublicKeys, v)
	}
	txn := functions.CreateTransaction(
		common2.TxVersion09,
		common2.NextTurnDPOSInfo,
		0,
		&nextTurnDPOSInfo,
		[]*common2.Attribute{},
		[]*common2.Input{},
		[]*common2.Output{},
		0,
		[]*program.Program{},
	)

	txn = CreateTransactionByType(txn, s.Chain)
	err, _ := txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:should not have next turn dpos info transaction")

	s.OriginalLedger.Arbitrators.SetNeedNextTurnDPOSInfo(true)
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:checkNextTurnDPOSInfoTransaction nextTurnDPOSInfo was wrong")

	s.Chain.GetState().DPoSV2ActiveHeight = 1
	err, _ = txn.SpecialContextCheck()
	s.EqualError(err,
		"transaction validate error: payload content invalid:checkNextTurnDPOSInfoTransaction nextTurnDPOSInfo was wrong")

}
