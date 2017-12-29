package httpnodeinfo

import (
	"Elastos.ELA/common/config"
	"Elastos.ELA/core/ledger"
	. "Elastos.ELA/net/protocol"
	"fmt"
	"html/template"
	"net/http"
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

type NgbNodeInfo struct {
	NgbId         string
	NgbAddr       string
	HttpInfoAddr  string
	HttpInfoPort  int
	HttpInfoStart bool
}

var node Noder

var templates = template.Must(template.New("info").Parse(page))

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var ngbrNodersInfo []NgbNodeInfo
	ngbrNoders := node.GetNeighborNoder()

	for i := 0; i < len(ngbrNoders); i++ {
		ngbHttpInfoAddr := ngbrNoders[i].GetAddr() + ":" + strconv.Itoa(ngbrNoders[i].GetHttpInfoPort())
		ngbrInfo := &NgbNodeInfo{
			NgbId:         fmt.Sprintf("0x%x", ngbrNoders[i].GetID()),
			NgbAddr:       ngbrNoders[i].GetAddr(),
			HttpInfoAddr:  ngbHttpInfoAddr,
			HttpInfoPort:  ngbrNoders[i].GetHttpInfoPort(),
			HttpInfoStart: ngbrNoders[i].GetHttpInfoState(),
		}
		ngbrNodersInfo = append(ngbrNodersInfo, *ngbrInfo)
	}

	pageInfo := &Info{
		NodeVersion:  config.Version,
		BlockHeight:  ledger.DefaultLedger.Blockchain.BlockHeight,
		NeighborCnt:  len(ngbrNoders),
		Neighbors:    ngbrNodersInfo,
		HttpRestPort: config.Parameters.HttpRestPort,
		HttpWsPort:   config.Parameters.HttpWsPort,
		HttpJsonPort: config.Parameters.HttpJsonPort,
		NodePort:     config.Parameters.NodePort,
		NodeId:       fmt.Sprintf("0x%x", node.GetID()),
	}

	err := templates.ExecuteTemplate(w, "info", pageInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartServer(n Noder) {
	node = n
	http.HandleFunc("/info", viewHandler)
	http.ListenAndServe(":"+strconv.Itoa(int(config.Parameters.HttpInfoPort)), nil)
}
