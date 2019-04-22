package outputpayload

import (
	"bytes"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/stretchr/testify/assert"
)

const (
	hash1 = "9dcad6d4ec2851bf522ddd301c7567caf98554a82a0bcce866de80b503909642"
)

var (
	ccfdata1, _ = common.Uint256FromHexString(hash1)
	cBytes      = []uint8{0, 157, 202, 214, 212, 236, 40, 81, 191, 82, 45, 221, 48,
		28, 117, 103, 202, 249, 133, 84, 168, 42, 11, 204, 232, 102, 222, 128,
		181, 3, 144, 150, 66}
)

func TestAdditionalFee_Serialize(t *testing.T) {
	ccf1 := AdditionalFee{
		Version: 0,
		TxHash:  *ccfdata1,
	}

	buf := new(bytes.Buffer)
	if err := ccf1.Serialize(buf); err != nil {
		t.Error("additional fee serialize failed")
	}
	if !bytes.Equal(buf.Bytes(), cBytes) {
		t.Error("additional fee serialize failed\n", common.BytesToHexString(buf.Bytes()))
	}
}

func TestAdditionalFee_Deserialize(t *testing.T) {
	buf := bytes.NewBuffer(cBytes)
	var ccf AdditionalFee
	if err := ccf.Deserialize(buf); err != nil {
		t.Error("additional fee deserialize failed")
	}
	if ccf.Version != 0 {
		t.Error("error version")
	}
	if common.BytesToHexString(ccf.TxHash.Bytes()) != hash1 {
		t.Error("error hash")
	}
}

func TestAdditionalFee_Validate(t *testing.T) {
	ccf2 := AdditionalFee{
		Version: 1,
		TxHash:  *ccfdata1,
	}
	err := ccf2.Validate()
	assert.EqualError(t, err, "invalid additional fee version")
}
