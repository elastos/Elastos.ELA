package types

type Vote_cr_info struct {
	Did            string `json:",omitempty"`
	Vote_type      string `json:",omitempty"`
	Txid           string `json:",omitempty"`
	N              int    `json:",omitempty"`
	Value          string `json:",omitempty"`
	Outputlock     int    `json:",omitempty"`
	Address        string `json:",omitempty"`
	Block_time     int64  `json:",omitempty"`
	Height         int64  `json:",omitempty"`
	Rank           int64  `json:",omitempty"`
	Candidate_info `json:",omitempty"`
	Is_valid       string `json:",omitempty"`
}

type Candidate_info struct {
	Code     string `json:",omitempty"`
	Did      string `json:",omitempty"`
	Nickname string `json:",omitempty"`
	Url      string `json:",omitempty"`
	Location int64  `json:",omitempty"`
	State    string `json:",omitempty"`
	Votes    string `json:",omitempty"`
	Index    int64
}

type Vote_cr_statistic_header struct {
	Value         string       `json:",omitempty"`
	Candidate_num int          `json:",omitempty"`
	Total_num     int          `json:",omitempty"`
	Txid          string       `json:",omitempty"`
	Height        int64        `json:",omitempty"`
	Candidates    []Candidates `json:",omitempty"`
	Block_time    int64        `json:",omitempty"`
	Is_valid      string       `json:",omitempty"`
}

type Candidates struct {
	Did   string `json:",omitempty"`
	Value string `json:",omitempty"`
}

type Vote_cr_statistic struct {
	Vote_Header Vote_cr_statistic_header `json:",omitempty"`
	Vote_Body   []Vote_cr_info           `json:",omitempty"`
}

type Vote_cr_statisticSorter []Vote_cr_statistic

func (a Vote_cr_statisticSorter) Len() int      { return len(a) }
func (a Vote_cr_statisticSorter) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a Vote_cr_statisticSorter) Less(i, j int) bool {
	return a[i].Vote_Header.Height > a[j].Vote_Header.Height
}

type Cr_voter struct {
	Address string
}
