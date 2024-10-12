// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"errors"
	"io"

	"github.com/elastos/Elastos.ELA/common"
)

const RecordSponsorVersion byte = 0x00

const SponsorMaxLength = 33

type RecordSponsor struct {
	Sponsor []byte
}

func (a *RecordSponsor) Data(version byte) []byte {
	//TODO: implement RegisterRecord.Data()
	return []byte{0}
}

// Serialize is the implement of SignableData interface.
func (a *RecordSponsor) Serialize(w io.Writer, version byte) error {
	err := common.WriteVarBytes(w, a.Sponsor)
	if err != nil {
		return errors.New("[RecordSponsor], Sponsor serialize failed.")
	}
	return nil
}

// Deserialize is the implement of SignableData interface.
func (a *RecordSponsor) Deserialize(r io.Reader, version byte) error {
	var err error
	a.Sponsor, err = common.ReadVarBytes(r, SponsorMaxLength,
		"payload record data")
	if err != nil {
		return errors.New("[RecordSponsor], Sponsor deserialize failed.")
	}
	return nil
}
