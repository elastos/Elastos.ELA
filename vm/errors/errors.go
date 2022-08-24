// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package errors

import "errors"

var (
	ErrBadValue = errors.New("bad value")
	ErrBadType  = errors.New("bad type")
	ErrOverLen  = errors.New("the count over the size")
	ErrFault    = errors.New("The exeution meet fault")
)
