// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"github.com/elastos/Elastos.ELA/database"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type TransactionProcessor interface {
	GetSaveProcessor() (database.TXProcessor, elaerr.ELAError)
	GetRollbackProcessor() (database.TXProcessor, elaerr.ELAError)
	//GetCreateProcessor() (database.TXProcessor, elaerr.ELAError)
}
