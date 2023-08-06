// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package settings

import (
	"github.com/RainFallsSilent/screw"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/transaction"
	"github.com/elastos/Elastos.ELA/core/types/functions"
	"github.com/elastos/Elastos.ELA/elanet/pact"
	"github.com/spf13/viper"
	"path/filepath"
	"strings"
)

type Settings struct {
	viper  *viper.Viper
	params *config.Configuration
}

func (s *Settings) Viper() *viper.Viper {
	return s.viper
}

// ignore the error, for command line
func (s *Settings) loadConfigFile(files string, cfg *config.Config) {
	paths, fileName := filepath.Split(files)
	fileExt := filepath.Ext(files)
	s.viper.AddConfigPath(paths)
	s.viper.SetConfigName(strings.TrimSuffix(fileName, fileExt))
	s.viper.SetConfigType(strings.TrimPrefix(fileExt, "."))
	if err := s.viper.ReadInConfig(); err != nil {
		return
	}

	crcArbiters := s.viper.Get("configuration.dposconfiguration.crcarbiters")
	if crcArbiters != nil {
		cfg.DPoSConfiguration.CRCArbiters = []string{}
	}

	s.viper.Unmarshal(&cfg)
}

func (s *Settings) SetupConfig(withScrew bool, about string, version string) *config.Configuration {
	// Initialize functions
	functions.GetTransactionByTxType = transaction.GetTransaction
	functions.GetTransactionByBytes = transaction.GetTransactionByBytes
	functions.CreateTransaction = transaction.CreateTransaction
	functions.GetTransactionParameters = transaction.GetTransactionparameters

	conf := &config.Config{
		Configuration: &config.DefaultParams,
	}
	if withScrew {
		screw.Bind(conf.Configuration, version, about)
	}
	if conf.Conf == "" {
		conf.Conf = config.ConfigFile
	}
	s.loadConfigFile(conf.Conf, conf)

	// switch activeNet params
	var testNet bool
	switch strings.ToLower(conf.ActiveNet) {
	case "testnet", "test":
		testNet = true
		conf.TestNet()
		s.loadConfigFile(conf.Conf, conf)
	case "regnet", "regtest", "reg":
		conf.RegNet()
		s.loadConfigFile(conf.Conf, conf)
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
		conf.Configuration = conf.InstantBlock()
	}
	if withScrew {
		screw.Bind(conf.Configuration, version, about)
	}
	conf.Configuration = conf.Sterilize()
	config.Parameters = conf.Configuration

	if len(conf.RpcConfiguration.WhiteIPList) > 1 {
		ips := strings.Split(conf.RpcConfiguration.WhiteIPList[0], ",")
		for i, ip := range ips {
			conf.RpcConfiguration.WhiteIPList[i] = ip
		}
	}
	return conf.Configuration
}

func NewSettings() *Settings {
	settings := &Settings{
		viper: viper.New(),
	}
	return settings
}
