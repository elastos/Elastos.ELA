package cli

import (
	"math/rand"
	"time"

	"Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	"Elastos.ELA/crypto"
)

func init() {
	log.Init()
	crypto.SetAlg(config.Parameters.EncryptAlg)
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
