// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"errors"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

type RecordTransaction struct {
	BaseTransaction
}

func (t *RecordTransaction) CheckTransactionPayload() error {
	switch t.Payload().(type) {
	case *payload.Record:
		return nil
	}

	return errors.New("invalid payload type")
}

func (t *RecordTransaction) IsAllowedInPOWConsensus() bool {
	return false
}
