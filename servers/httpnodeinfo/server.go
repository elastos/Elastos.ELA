package httpnodeinfo

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/elastos/Elastos.ELA/blockchain"
	"github.com/elastos/Elastos.ELA/config"
	"github.com/elastos/Elastos.ELA/node"
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
	NodePort      uint16
	NodeID        string
}

type NgbNodeInfo struct {
	NgbID   string
	NbrAddr string
}

var templates = template.Must(template.New("info").Parse(page))

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var ngbrNodersInfo []NgbNodeInfo

	neighbors := node.GetNeighborNodes()

	for i := 0; i < len(neighbors); i++ {
		ngbrNodersInfo = append(ngbrNodersInfo, NgbNodeInfo{
			NgbID:   fmt.Sprintf("0x%x", neighbors[i].ID()),
			NbrAddr: neighbors[i].Addr(),
		})
	}

	pageInfo := &Info{
		BlockHeight:  blockchain.DefaultLedger.Blockchain.BlockHeight,
		NeighborCnt:  len(neighbors),
		Neighbors:    ngbrNodersInfo,
		HttpRestPort: config.Parameters.HttpRestPort,
		HttpWsPort:   config.Parameters.HttpWsPort,
		HttpJsonPort: config.Parameters.HttpJsonPort,
		NodePort:     config.Parameters.NodePort,
		NodeID:       fmt.Sprintf("0x%x", node.LocalNode.ID()),
	}

	err := templates.ExecuteTemplate(w, "info", pageInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartServer() {
	http.HandleFunc("/info", viewHandler)
	http.ListenAndServe(":"+strconv.Itoa(int(config.Parameters.HttpInfoPort)), nil)
}
