package httpnodeinfo

import (
	"Elastos.ELA/common/config"
	"Elastos.ELA/core/ledger"
	. "Elastos.ELA/net/protocol"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
)

type Info struct {
	NodeVersion   string
	BlockHeight   uint32
	NeighborCnt   int
	Neighbors     []NgbNodeInfo
	HttpRestPort  int
	HttpWsPort    int
	HttpJsonPort  int
	HttpLocalPort int
	NodePort      int
	NodeId        string
}

var node Noder

var templates = template.Must(template.New("info").Parse(page))

func newNgbNodeInfo(ngbId string, ngbAddr string, httpInfoAddr string, httpInfoPort int, httpInfoStart bool) *NgbNodeInfo {
	return &NgbNodeInfo{NgbId: ngbId, NgbAddr: ngbAddr, HttpInfoAddr: httpInfoAddr,
		HttpInfoPort: httpInfoPort, HttpInfoStart: httpInfoStart}
}

func initPageInfo(blockHeight uint32, ngbrCnt int, ngbrsInfo []NgbNodeInfo) (*Info) {
	id := fmt.Sprintf("0x%x", node.GetID())
	return &Info{
		NodeVersion: config.Version,
		BlockHeight: blockHeight,
		NeighborCnt: ngbrCnt,
		Neighbors: ngbrsInfo,
		HttpRestPort:  config.Parameters.HttpRestPort,
		HttpWsPort:    config.Parameters.HttpWsPort,
		HttpJsonPort:  config.Parameters.HttpJsonPort,
		NodePort:      config.Parameters.NodePort,
		NodeId:        id }
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var ngbrNodersInfo []NgbNodeInfo
	var ngbId string
	var ngbAddr string
	var ngbInfoPort int
	var ngbInfoState bool
	var ngbHttpInfoAddr string


	ngbrNoders := node.GetNeighborNoder()
	ngbrsLen := len(ngbrNoders)
	for i := 0; i < ngbrsLen; i++ {

		ngbAddr = ngbrNoders[i].GetAddr()
		ngbInfoPort = ngbrNoders[i].GetHttpInfoPort()
		ngbInfoState = ngbrNoders[i].GetHttpInfoState()
		ngbHttpInfoAddr = ngbAddr + ":" + strconv.Itoa(ngbInfoPort)
		ngbId = fmt.Sprintf("0x%x", ngbrNoders[i].GetID())

		ngbrInfo := newNgbNodeInfo(ngbId, ngbAddr, ngbHttpInfoAddr, ngbInfoPort, ngbInfoState)
		ngbrNodersInfo = append(ngbrNodersInfo, *ngbrInfo)
	}
	sort.Sort(NgbNodeInfoSlice(ngbrNodersInfo))

	blockHeight := ledger.DefaultLedger.Blockchain.BlockHeight
	pageInfo := initPageInfo(blockHeight, ngbrsLen, ngbrNodersInfo)


	err := templates.ExecuteTemplate(w, "info", pageInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartServer(n Noder) {
	node = n
	port := int(config.Parameters.HttpInfoPort)
	http.HandleFunc("/info", viewHandler)
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
