// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package settings

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/elanet/pact"

	// todo fork it to Elastos
	"github.com/RainFallsSilent/screw"
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

	crcArbiters := s.viper.Get("configuration.dposconfiguration.crcarbiters")
	if crcArbiters != nil {
		cfg.DPoSConfiguration.CRCArbiters = []string{}
	}

	if err := s.viper.Unmarshal(&cfg); err != nil {
		return &config.DefaultParams, errors.New("configuration files can't be loaded" + err.Error())
	}
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
		Configuration: &config.DefaultParams,
	}
	// set mainNet params
	conf, _ := s.loadConfigFile(configFile, params)

	// switch activeNet params
	var testNet bool
	switch strings.ToLower(conf.ActiveNet) {
	case "testnet", "test":
		testNet = true
		testnet := config.Config{
			Configuration: params.TestNet(),
		}
		conf, _ = s.loadConfigFile(configFile, testnet)
	case "regnet", "regtest", "reg":
		regnet := config.Config{
			Configuration: params.RegNet(),
		}
		conf, _ = s.loadConfigFile(configFile, regnet)
	}

	if conf.MaxBlockSize > 0 {
		pact.MaxBlockContextSize = conf.MaxBlockSize
	} else if !testNet {
		pact.MaxBlockContextSize = 2000000
	}

	if conf.MaxBlockHeaderSize > 0 {
		pact.MaxBlockHeaderSize = conf.MaxBlockHeaderSize
	}

	if conf.MaxTxPerBlock > 0 {
		pact.MaxTxPerBlock = conf.MaxTxPerBlock
	} else {
		pact.MaxTxPerBlock = 10000
	}

	instantBlock := conf.PowConfiguration.InstantBlock
	if instantBlock {
		conf = conf.InstantBlock()
	}
	screw.Bind(conf)
	conf = conf.Sterilize()
	config.Parameters = conf
	return conf
}

func NewSettings() *Settings {
	settings := &Settings{
		viper: viper.New(),
	}
	return settings
}
