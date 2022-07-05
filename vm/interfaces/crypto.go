// Copyright (c) 2017-2022 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package interfaces

type ICrypto interface {
	Hash168(data []byte) []byte

	Hash256(data []byte) []byte

	VerifySignature(data []byte, signature []byte, pubkey []byte) error
}
