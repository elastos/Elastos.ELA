package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/elastos/Elastos.ELA/cmd/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/common/log"
)

const configFilename = "./config.json"

func init() {
	var printLevel uint8
	var maxPerLogSize int64
	var maxLogsSize int64
	var jsonRPCPort int
	file, err := ioutil.ReadFile(configFilename)
	if err != nil {
		fmt.Println("read config file error:", err)
		printLevel = config.Template.PrintLevel
		maxLogsSize = config.Template.MaxLogsSize
		maxPerLogSize = config.Template.MaxPerLogSize
		jsonRPCPort = config.Template.HttpJsonPort

	} else {
		// Remove the UTF-8 Byte Order Mark
		file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

		var cfgFile config.ConfigFile
		if err := json.Unmarshal(file, &cfgFile); err != nil {
			fmt.Println("unmarshal config file error:", err)
			os.Exit(1)
		}
		printLevel = cfgFile.PrintLevel
		maxLogsSize = cfgFile.MaxLogsSize
		maxPerLogSize = cfgFile.MaxPerLogSize
		jsonRPCPort = cfgFile.HttpJsonPort
	}

	common.JsonRPCPort = jsonRPCPort
	log.NewDefault(
		printLevel,
		maxPerLogSize,
		maxLogsSize,
	)
	//seed transaction nonce
	rand.Seed(time.Now().UnixNano())
}
