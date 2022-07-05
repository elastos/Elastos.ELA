// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package vm

type VMState byte

const (
	NONE  VMState = 0
	HALT  VMState = 1 << 0
	FAULT VMState = 1 << 1
	BREAK VMState = 1 << 2

	INSUFFICIENT_RESOURCE VMState = 1 << 4
)
