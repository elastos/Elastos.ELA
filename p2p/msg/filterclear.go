// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package msg

import (
	"github.com/elastos/Elastos.ELA/p2p"
)

// Ensure FilterClear implement p2p.Message interface.
var _ p2p.Message = (*FilterClear)(nil)

type FilterClear struct{ empty }

func (msg *FilterClear) CMD() string {
	return p2p.CmdFilterClear
}
