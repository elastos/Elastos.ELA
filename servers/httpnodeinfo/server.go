package httpnodeinfo

import (
	"encoding/hex"
	"fmt"
	"time"

	"html/template"
	"net/http"
	"strconv"

	chain "github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/dpos/account"
	"github.com/elastos/Elastos.ELA/dpos/p2p/peer"
	"github.com/elastos/Elastos.ELA/servers"
)

var act account.Account

type Info struct {
	NodeVersion   string
	BlockHeight   uint32
	NeighborCnt   int
	Neighbors     []NgbNodeInfo
	HttpRestPort  int
	HttpWsPort    int
	HttpJsonPort  int
	HttpLocalPort int
	NodePort      uint16
	IsProducer    bool
	BstBlockTime  string
	ProducerInfo
}

type NgbNodeInfo struct {
	NgbID   string
	NbrAddr string
}

type ProducerInfo struct {
	NodePublicKey      string
	OwnerPublicKey     string
	State              string
	Votes              common.Fixed64
	DPosNeighbors      []DPosNeighbourInfo
	LastProposalHeight uint32
}

type DPosNeighbourInfo struct {
	PID   peer.PID
	Addr  string
	State string
}

var templates = template.Must(template.New("info").Parse(page))

func viewHandler(w http.ResponseWriter, r *http.Request) {
	arbiter := servers.Arbiter
	var ngbrNodersInfo []NgbNodeInfo

	peers := servers.Server.ConnectedPeers()

	for _, ip := range peers {
		p := ip.ToPeer()
		ngbrNodersInfo = append(ngbrNodersInfo, NgbNodeInfo{
			NgbID:   fmt.Sprintf("0x%x", p.ID()),
			NbrAddr: p.Addr(),
		})
	}
	BestBlock, _ := chain.DefaultLedger.Blockchain.GetBlockByHash(chain.DefaultLedger.Blockchain.CurrentBlockHash())
	var httpRestPort = config.Parameters.HttpRestPort
	var httpWsPort = config.Parameters.HttpWsPort
	var httpJsonPort = config.Parameters.HttpJsonPort
	var nodePort = config.Parameters.NodePort
	var netType = config.Parameters.ActiveNet
	var index int
	switch netType {
	case "mainnet", "":
		index = 0
	case "regnet":
		index = 2
	}

	if config.Parameters.HttpRestPort == 0 {
		httpRestPort = 20330 + index * 1000 + 4
	}
	if config.Parameters.HttpJsonPort == 0 {
		httpJsonPort = 20330 + index * 1000 + 6
	}
	if config.Parameters.HttpWsPort == 0 {
		httpWsPort = 20330 + index * 1000 + 5
	}
	if config.Parameters.NodePort == 0 {
		nodePort = uint16(20330 + index * 1000 + 8)
	}
	pageInfo := &Info{
		BlockHeight:  chain.DefaultLedger.Blockchain.GetHeight(),
		NeighborCnt:  len(peers),
		Neighbors:    ngbrNodersInfo,
		HttpRestPort: httpRestPort,
		HttpWsPort:   httpWsPort,
		HttpJsonPort: httpJsonPort,
		NodePort:     nodePort,
		BstBlockTime: time.Unix(int64(BestBlock.Timestamp), 0).String(),
		IsProducer:   arbiter != nil,
	}

	if pageInfo.IsProducer {
		for _, v := range arbiter.GetArbiterPeersInfo() {
			pageInfo.DPosNeighbors = append(pageInfo.DPosNeighbors, DPosNeighbourInfo{
				v.PID, v.Addr, v.State.String(),
			})
		}
		//pageInfo.DPosNeighbors = arbiter.GetArbiterPeersInfo()
		pageInfo.NodePublicKey = hex.EncodeToString(act.PublicKeyBytes())
		producers := servers.Chain.GetState().GetAllProducers()
		for _, producer := range producers {
			if string(producer.NodePublicKey()) == string(act.PublicKeyBytes()) {
				pageInfo.OwnerPublicKey = hex.EncodeToString(producer.OwnerPublicKey())
				pageInfo.State = producer.State().String()
				pageInfo.Votes = producer.Votes()
			}
		}
	}

	err := templates.ExecuteTemplate(w, "info", pageInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartServer(a account.Account) {
	act = a
	http.HandleFunc("/info", viewHandler)
	http.ListenAndServe(":"+strconv.Itoa(int(config.Parameters.HttpInfoPort)), nil)
}
