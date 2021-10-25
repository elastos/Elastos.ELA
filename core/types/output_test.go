// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package types

import (
	"bytes"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
)

const (
	OUTPUTHEX = "a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0a08601000000000000000000210fcd528848be05f8cffe5d99896c44bdeec70502010001000103010203"
)

var (
	assetID, _   = common.Uint256FromHexString("a3d0eaa466df74983b5d7c543de6904f4c9418ead5ffd6d25814234a96db37b0")
	recipient, _ = common.Uint168FromAddress("EJbTbWd8a9rdutUfvBxhcrvEeNy21tW1Ee")
)

func TestOutput_Serialize(t *testing.T) {
	// C0
	output := common2.Output{
		AssetID:     *assetID,
		Value:       100000,
		OutputLock:  0,
		ProgramHash: *recipient,
		Type:        common2.OTVote,
		Payload: &outputpayload.VoteOutput{
			Version: 0,
			Contents: []outputpayload.VoteContent{
				outputpayload.VoteContent{
					VoteType: 0,
					CandidateVotes: []outputpayload.CandidateVotes{
						{[]byte{1, 2, 3}, 0},
					},
				},
			},
		},
	}

	buf := new(bytes.Buffer)
	if err := output.Serialize(buf, common2.TxVersion09); err != nil {
		t.Error("output serialize failed")
	}

	resBytes, _ := common.HexStringToBytes(OUTPUTHEX)
	if !bytes.Equal(buf.Bytes(), resBytes) {
		t.Error("output serialize failed\n", common.BytesToHexString(buf.Bytes()))
	}
}

func TestOutput_Deserialize(t *testing.T) {
	// C0
	outputBytes, _ := common.HexStringToBytes(OUTPUTHEX)
	outputBuf := bytes.NewBuffer(outputBytes)

	var output common2.Output
	if err := output.Deserialize(outputBuf, common2.TxVersion09); err != nil {
		t.Error("output deserialize failed")
	}

	buf := new(bytes.Buffer)
	if err := output.Serialize(buf, common2.TxVersion09); err != nil {
		t.Error("output serialize failed")
	}

	resBytes, _ := common.HexStringToBytes(OUTPUTHEX)
	if !bytes.Equal(buf.Bytes(), resBytes) {
		t.Error("output deserialize failed")
	}
}
