package cli

import (
	"math/rand"
	"time"

	"DNA_POW/common/config"
	"DNA_POW/common/log"
	"DNA_POW/crypto"
)

func init() {
	log.Init()
	crypto.SetAlg(config.Parameters.EncryptAlg)
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
