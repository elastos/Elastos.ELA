// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package events

import (
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/utils/test"
	"github.com/stretchr/testify/assert"
)

func TestNotify2(t *testing.T) {
	// 创建一个 big.Int 对象
	intValue := big.NewInt(10000)

	// 转换为 big.Float
	floatValue := new(big.Float).SetInt(intValue)
	decimal := int(3)
	floatValue.Quo(floatValue, new(big.Float).SetFloat64(math.Pow(float64(10), float64(decimal))))

	// 将 big.Float 格式化为字符串
	formattedValue := floatValue.Text('f', 18)

	fmt.Println(formattedValue)

}

func TestNotify(t *testing.T) {
	test.SkipShort(t)
	notifyChan := make(chan struct{})
	Subscribe(func(event *Event) {
		notifyChan <- struct{}{}
	})

	for i := 0; i < 100; i++ {
		go func() {
			Notify(ETBlockAccepted, nil)
		}()
	}

	for i := 0; i < 100; i++ {
		select {
		case <-notifyChan:
		case <-time.After(time.Millisecond):
			t.Error("notify timeout")
		}
	}
}

func TestRecursiveNotify(t *testing.T) {
	Subscribe(func(event *Event) {
		Notify(ETBlockConnected, nil)
	})

	go func() {
		defer func() {
			if err := recover(); err != nil {
				if !assert.Equal(t, err, "recursive notifies detected") {
					t.FailNow()
				}
			}
		}()
		Notify(ETBlockAccepted, nil)
	}()

	<-time.After(time.Millisecond)

}
