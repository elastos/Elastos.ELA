// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

//func TestCoinBase_FunctionRewrite(t *testing.T) {
//	var para = &interfaces.CheckParameters{
//		Version:        0,
//		TxType:         0,
//		PayloadVersion: 0,
//		Attributes:     nil,
//		Inputs: []*common2.Input{
//			{
//				Previous: common2.OutPoint{
//					TxID:  common.Uint256{1},
//					Index: 0,
//				},
//				Sequence: 0,
//			},
//		},
//		Outputs: []*common2.Output{
//			{
//				AssetID:     common.Uint256{2},
//				Value:       100,
//				OutputLock:  0,
//				ProgramHash: common.Uint168{},
//				Type:        0,
//				Payload:     nil,
//			},
//		},
//		LockTime:               0,
//		Programs:               nil,
//		TxHash:                 common.Uint256{},
//		BlockHeight:            0,
//		CRCommitteeStartHeight: 0,
//		ConsensusAlgorithm:     0,
//		DestroyELAAddress:      common.Uint168{},
//		CRAssetsAddress:        common.Uint168{},
//		FoundationAddress:      common.Uint168{},
//	} // self check
//	payload := CoinBase{
//		DefaultChecker: DefaultChecker{
//			IsTxHashDuplicateFunction: func(txhash common.Uint256) bool { return false },
//			GetTxReferenceFunction: func(para interfaces.CheckParameters) (map[*common2.Input]common2.Output, error) {
//				result := make(map[*common2.Input]common2.Output)
//				return result, nil
//			},
//		},
//	}
//	payload.ContextCheck(para)
//	payload.SpecialContextCheck(para)
//
//	// default check
//	payload2 := Confirm{
//		DefaultChecker: DefaultChecker{
//			IsTxHashDuplicateFunction: func(txhash common.Uint256) bool { return false },
//			GetTxReferenceFunction: func(para interfaces.CheckParameters) (map[*common2.Input]common2.Output, error) {
//				result := make(map[*common2.Input]common2.Output)
//				return result, nil
//			},
//		},
//	}
//	payload2.ContextCheck(para)
//	payload2.SpecialContextCheck(para)
//
//}
