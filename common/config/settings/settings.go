// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package settings

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"

	"github.com/fungolang/screw"
	"github.com/spf13/viper"
)

type Settings struct {
	viper  *viper.Viper
	params *config.Configuration
}

func (s *Settings) Viper() *viper.Viper {
	return s.viper
}

func (s *Settings) loadConfigFile(files string, cfg config.Config) (*config.Configuration, error) {
	paths, fileName := filepath.Split(files)
	fileExt := filepath.Ext(files)
	s.viper.AddConfigPath("./" + paths)
	s.viper.SetConfigName(strings.TrimSuffix(fileName, fileExt))
	s.viper.SetConfigType(strings.TrimPrefix(fileExt, "."))
	if err := s.viper.ReadInConfig(); err != nil {
		return &config.DefaultParams, errors.New("cannot read configuration" + err.Error())
	}
	if err := s.viper.Unmarshal(&cfg); err != nil {
		return &config.DefaultParams, errors.New("configuration files can't be loaded" + err.Error())
	}
	//for _, key := range s.viper.AllKeys() {
	//	fmt.Printf("---%s = %s\n", key, s.viper.Get(key))
	//}
	return cfg.Configuration, nil
}

func (s *Settings) SetupConfig() *config.Configuration {
	// Initialize functions
	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters

	configFile := config.ConfigFile
	params := config.Config{
		Configuration: config.GetDefaultParams(),
	}
	// set mainNet params
	conf, err := s.loadConfigFile(configFile, params)
	if err != nil {
		fmt.Println("loadConfigFile err", err.Error())
	}

	// switch activeNet params
	switch strings.ToLower(conf.ActiveNet) {
	case "testnet", "test":
		testnet := config.Config{
			Configuration: params.TestNet(),
		}
		conf, err = s.loadConfigFile(configFile, testnet)
	case "regnet", "regtest", "reg":
		regnet := config.Config{
			Configuration: params.RegNet(),
		}
		conf, err = s.loadConfigFile(configFile, regnet)
	}

	screw.Bind(conf)
	instantBlock := conf.PowConfiguration.InstantBlock
	if instantBlock {
		conf = conf.InstantBlock()
	}
	config.Parameters = conf
	return conf
}

func NewSettings() *Settings {
	settings := &Settings{
		viper: viper.New(),
	}
	return settings
}
