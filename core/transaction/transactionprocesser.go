// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package transaction

import (
	"github.com/elastos/Elastos.ELA/database"
	elaerr "github.com/elastos/Elastos.ELA/errors"
)

type DefaultProcessor struct {
}

func (t *DefaultProcessor) GetSaveProcessor() (database.TXProcessor, elaerr.ELAError) {
	return nil, nil
}

func (t *DefaultProcessor) GetRollbackProcessor() (database.TXProcessor, elaerr.ELAError) {
	return nil, nil
}
