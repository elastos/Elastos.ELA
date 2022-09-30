// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"bytes"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"math/rand"
	"strconv"
)

func (s *transactionSuite) TestRecord_SerializeDeserialize() {
	txn := randomOldVersionTransaction(true, byte(common2.Record), s.InputNum, s.OutputNum, s.AttrNum, s.ProgramNum)
	txn.SetPayload(&payload.Record{
		Type:    "test record type",
		Content: []byte(strconv.FormatUint(rand.Uint64(), 10)),
	})

	serializedData := new(bytes.Buffer)
	txn.Serialize(serializedData)

	txn2, err := functions.GetTransactionByBytes(serializedData)
	if err != nil {
		s.Assert()
	}
	txn2.Deserialize(serializedData)

	assertOldVersionTxEqual(true, &s.Suite, txn, txn2, s.InputNum, s.OutputNum, s.AttrNum, s.ProgramNum)

	p1 := txn.Payload().(*payload.Record)
	p2 := txn2.Payload().(*payload.Record)

	s.Equal(p1.Type, p2.Type)
	s.True(bytes.Equal(p1.Content, p2.Content))
}
