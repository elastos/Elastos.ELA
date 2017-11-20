package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const (
	DefaultConfigFilename = "./config.json"
	MINGENBLOCKTIME       = 2
	DEFAULTGENBLOCKTIME   = 6
)

var Version string

type PowConfiguration struct {
	Switch             string `json:"Switch"`
	PayToAddr          string `json:"PayToAddr"`
	MiningServerIP     string `josn:"MiningServerIP"`
	MiningServerPort   int    `josn:"MiningServerPort"`
	MiningSelfPort     int    `josn:"MiningSelfPort"`
	WalletVersion      int    `json:"WalletVersion"`
	ProtocolVersion    int    `json:"ProtocolVersion"`
	TestNet            bool   `json:"testnet"`
	Proxy              string `json:"Proxy"`
	CoMining           bool   `json:"CoMining"`
	AutoMining         bool   `json:"AutoMining"`
	MinerInfo          string `json:"MinerInfo"`
	TargetTimeSpan     int64  `json:"TargetTimeSpan"`
	TargetTimePerBlock int64  `json:"TargetTimePerBlock"`
	PowLimitBits       uint32 `json:"PowLimitBits"`
}

type Configuration struct {
	Magic            uint32           `json:"Magic"`
	Version          int              `json:"Version"`
	SeedList         []string         `json:"SeedList"`
	BookKeepers      []string         `json:"BookKeepers"` // The default book keepers' publickey
	HttpRestPort     int              `json:"HttpRestPort"`
	RestCertPath     string           `json:"RestCertPath"`
	RestKeyPath      string           `json:"RestKeyPath"`
	HttpInfoPort     uint16           `json:"HttpInfoPort"`
	HttpInfoStart    bool             `json:"HttpInfoStart"`
	HttpWsPort       int              `json:"HttpWsPort"`
	HttpJsonPort     int              `json:"HttpJsonPort"`
	HttpLocalPort    int              `json:"HttpLocalPort"`
	OauthServerUrl   string           `json:"OauthServerUrl"`
	NoticeServerUrl  string           `json:"NoticeServerUrl"`
	NodePort         int              `json:"NodePort"`
	NodeType         string           `json:"NodeType"`
	WebSocketPort    int              `json:"WebSocketPort"`
	PrintLevel       int              `json:"PrintLevel"`
	IsTLS            bool             `json:"IsTLS"`
	CertPath         string           `json:"CertPath"`
	KeyPath          string           `json:"KeyPath"`
	CAPath           string           `json:"CAPath"`
	GenBlockTime     uint             `json:"GenBlockTime"`
	MultiCoreNum     uint             `json:"MultiCoreNum"`
	EncryptAlg       string           `json:"EncryptAlg"`
	MaxLogSize       int64            `json:"MaxLogSize"`
	MaxTxInBlock     int              `json:"MaxTransactionInBlock"`
	MaxBlockSize     int              `json:"MaxBlockSize"`
	ConsensusType    string           `json:"ConsensusType"`
	PowConfiguration PowConfiguration `json:"PowConfiguration"`
	MaxHdrSyncReqs   int              `json:"MaxConcurrentSyncHeaderReqs"`
	DefaultMaxPeers  uint             `json:"DefaultMaxPeers"`
	GetAddrMax       uint             `json:"GetAddrMax"`
	MaxOutboundCnt   uint             `json:"MaxOutboundCnt"`
	//AddCheckpoints format: "<height>:<hash>"
	AddCheckpoints []string `json:"AddCheckpoints"`
}

type ConfigFile struct {
	ConfigFile Configuration `json:"Configuration"`
}

var Parameters *Configuration

func init() {
	file, e := ioutil.ReadFile(DefaultConfigFilename)
	if e != nil {
		log.Fatalf("File error: %v\n", e)
		os.Exit(1)
	}
	// Remove the UTF-8 Byte Order Mark
	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

	config := ConfigFile{}
	e = json.Unmarshal(file, &config)
	if e != nil {
		log.Fatalf("Unmarshal json file erro %v", e)
		os.Exit(1)
	}
	Parameters = &(config.ConfigFile)
	fmt.Println(" Parameters.AddCheckpoints ", Parameters.AddCheckpoints)

}
