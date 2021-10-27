// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"github.com/elastos/Elastos.ELA/common"
	"io"
)

type Transaction interface {
	PayloadChecker



	String() string
	Serialize(w io.Writer) error
	SerializeUnsigned(w io.Writer) error
	Deserialize(r io.Reader) error
	DeserializeUnsigned(r io.Reader) error
	GetSize() int
	Hash() common.Uint256

}
