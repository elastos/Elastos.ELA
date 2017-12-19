package cli

import (
	"math/rand"
	"time"

	"ELA/common/config"
	"ELA/common/log"
	"ELA/crypto"
)

func init() {
	log.Init()
	crypto.SetAlg(config.Parameters.EncryptAlg)
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
