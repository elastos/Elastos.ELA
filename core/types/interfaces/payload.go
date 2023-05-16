// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

import (
	"io"
)

// Payload define the func for loading the payload data
// base on payload type which have different structure
type Payload interface {
	// Get payload data
	Data(version byte) []byte

	Serialize(w io.Writer, version byte) error

	Deserialize(r io.Reader, version byte) error

}
