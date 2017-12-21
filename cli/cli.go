package cli

import (
	"math/rand"
	"time"
	"Elastos.ELA/common/log"

)

func init() {
	log.Init()
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
