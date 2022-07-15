package common

type Vote_Info struct {
	Producer_public_key string `json:",omitempty"`
	Vote_type           string `json:",omitempty"`
	Txid                string `json:",omitempty"`
	N                   int    `json:",omitempty"`
	Value               string `json:",omitempty"`
	Outputlock          int    `json:",omitempty"`
	Address             string `json:",omitempty"`
	Block_time          int64  `json:",omitempty"`
	Height              int64  `json:",omitempty"`
	Rank                int64  `json:",omitempty"`
	Producer_Info       `json:",omitempty"`
	Is_valid            string `json:",omitempty"`
	Reward              string `json:",omitempty"`
	EstRewardPerYear    string `json:",omitempty"`
}

type Vote_StatisticHeader struct {
	Value      string   `json:",omitempty"`
	Node_num   int      `json:",omitempty"`
	Txid       string   `json:",omitempty"`
	Height     int64    `json:",omitempty"`
	Nodes      []string `json:",omitempty"`
	Block_time int64    `json:",omitempty"`
	Is_valid   string   `json:",omitempty"`
}

type Vote_Statistic struct {
	Vote_Header Vote_StatisticHeader `json:",omitempty"`
	Vote_Body   []Vote_Info          `json:",omitempty"`
}

type Vote_Statisticsorter []Vote_Statistic

func (a Vote_Statisticsorter) Len() int      { return len(a) }
func (a Vote_Statisticsorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a Vote_Statisticsorter) Less(i, j int) bool {
	return a[i].Vote_Header.Height > a[j].Vote_Header.Height
}

type Producer_Info struct {
	OwnerPublickey string
	NodePublickey  string
	Nickname       string
	Url            string
	Location       int64
	Active         int
	Votes          string
	Netaddress     string
	State          string
	Registerheight int64
	Cancelheight   int64
	Inactiveheight int64
	Illegalheight  int64
	Index          int64
}
