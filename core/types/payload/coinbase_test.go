// Copyright (c) 2017-2021 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import "testing"

func TestCoinBase_FunctionRewrite(t *testing.T) {
	// self check
	payload := CoinBase{}
	payload.SpecialCheck(&CheckParameters{})

	// default check
	payload2 := Confirm{}
	payload2.SpecialCheck(&CheckParameters{})
}
