package blockchain

import (
	"bytes"
	"strconv"
	"strings"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/log"
	common2 "github.com/elastos/Elastos.ELA/core/types/common"
	"github.com/elastos/Elastos.ELA/core/types/interfaces"
	"github.com/elastos/Elastos.ELA/core/types/outputpayload"
	"github.com/elastos/Elastos.ELA/core/types/payload"
)

var MemPoolEx MemPool

type MemPool struct {
	i    int
	c    IChainStoreExtend
	is_p map[common.Uint256]bool
	p    map[string][]byte
	l    sync.RWMutex
}

func (m *MemPool) AppendToMemPool(tx interfaces.Transaction) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Recovered from AppendToMemPool ", r)
		}
	}()
	m.l.RLock()
	if _, ok := m.is_p[tx.Hash()]; ok {
		m.l.RUnlock()
		return nil
	}
	m.l.RUnlock()
	txhs := make([]common2.TransactionHistory, 0)

	var memo []byte
	var txType = tx.TxType()
	for _, attr := range tx.Attributes() {
		if attr.Usage == common2.Memo {
			memo = attr.Data
		}
	}

	isCrossTx := false
	if txType == common2.TransferCrossChainAsset {
		isCrossTx = true
	}
	voteType := m.isVoteTx(tx)

	spend := make(map[common.Uint168]int64)
	var totalInput int64 = 0
	var from []common.Uint168
	var to []common.Uint168
	for _, input := range tx.Inputs() {
		txid := input.Previous.TxID
		index := input.Previous.Index
		referTx, _, err := m.c.GetTransaction(txid)
		if err != nil {
			return err
		}
		address := referTx.Outputs()[index].ProgramHash
		totalInput += int64(referTx.Outputs()[index].Value)
		v, ok := spend[address]
		if ok {
			spend[address] = v + int64(referTx.Outputs()[index].Value)
		} else {
			spend[address] = int64(referTx.Outputs()[index].Value)
		}
		if !common.ContainsU168(address, from) {
			from = append(from, address)
		}
	}
	receive := make(map[common.Uint168]int64)
	var totalOutput int64 = 0
	for _, output := range tx.Outputs() {
		address, _ := output.ProgramHash.ToAddress()
		var valueCross int64
		if isCrossTx == true && (output.ProgramHash == MINING_ADDR || strings.Index(address, "X") == 0 || address == "4oLvT2") {
			switch pl := tx.Payload().(type) {
			case *payload.TransferCrossChainAsset:
				valueCross = int64(pl.CrossChainAmounts[0])
			}
		}
		if valueCross != 0 {
			totalOutput += valueCross
		} else {
			totalOutput += int64(output.Value)
		}
		v, ok := receive[output.ProgramHash]
		if ok {
			receive[output.ProgramHash] = v + int64(output.Value)
		} else {
			receive[output.ProgramHash] = int64(output.Value)
		}
		if !common.ContainsU168(output.ProgramHash, to) {
			to = append(to, output.ProgramHash)
		}

	}
	fee := totalInput - totalOutput
	for k, r := range receive {
		transferType := RECEIVED
		s, ok := spend[k]
		var value int64
		if ok {
			if s > r {
				value = s - r
				transferType = SENT
			} else {
				value = r - s
			}
			delete(spend, k)
		} else {
			value = r
		}
		var realFee = common.Fixed64(fee)
		var rto = to
		if transferType == RECEIVED {
			realFee = 0
			rto = []common.Uint168{k}
		}

		if transferType == SENT {
			from = []common.Uint168{k}
		}

		txh := common2.TransactionHistory{}
		txh.Value = common.Fixed64(value)
		txh.Address = k
		txh.Inputs = from
		txh.TxType = txType
		txh.VoteType = voteType
		txh.Txid = tx.Hash()
		txh.Height = 0
		txh.Time = 0
		txh.Type = []byte(transferType)
		txh.Fee = realFee
		if len(rto) > 10 {
			txh.Outputs = rto[0:10]
		} else {
			txh.Outputs = rto
		}
		txh.Memo = memo
		txh.Status = 1
		txhs = append(txhs, txh)
	}

	for k, r := range spend {
		txh := common2.TransactionHistory{}
		txh.Value = common.Fixed64(r)
		txh.Address = k
		txh.Inputs = []common.Uint168{k}
		txh.TxType = txType
		txh.VoteType = voteType
		txh.Txid = tx.Hash()
		txh.Height = 0
		txh.Time = 0
		txh.Type = []byte(SENT)
		txh.Fee = common.Fixed64(fee)
		if len(to) > 10 {
			txh.Outputs = to[0:10]
		} else {
			txh.Outputs = to
		}
		txh.Memo = memo
		txh.Status = 1
		txhs = append(txhs, txh)
	}
	for _, p := range txhs {
		m.l.Lock()
		m.is_p[p.Txid] = true
		m.l.Unlock()
		m.i += m.i
		err := m.store(p.Txid, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MemPool) isVoteTx(tx interfaces.Transaction) (vt common2.VoteCategory) {
	version := tx.Version()
	if version == 0x09 {
		vout := tx.Outputs()
		for _, v := range vout {
			if v.Type == 0x01 && v.AssetID == *ELA_ASSET {
				outputPayload, ok := v.Payload.(*outputpayload.VoteOutput)
				if !ok || outputPayload == nil {
					continue
				}
				contents := outputPayload.Contents
				for _, cv := range contents {
					votetype := cv.VoteType
					switch votetype {
					case 0x00:
						vt = vt | common2.DPoS
					case 0x01:
						vt = vt | common2.CRC
					case 0x02:
						vt = vt | common2.Proposal
					case 0x03:
						vt = vt | common2.Impeachment
					}
				}
			}
		}
	}
	return
}

func (m *MemPool) store(txid common.Uint256, history common2.TransactionHistory) error {
	m.l.Lock()
	defer m.l.Unlock()
	addr, _ := history.Address.ToAddress()
	value := new(bytes.Buffer)
	history.Serialize(value)
	m.p[addr+txid.String()+strconv.Itoa(m.i)] = value.Bytes()
	return nil
}

func (m *MemPool) GetMemPoolTx(address *common.Uint168) (ret []common2.TransactionHistoryDisplay) {
	m.l.RLock()
	defer m.l.RUnlock()
	for k, v := range m.p {
		addr, err := address.ToAddress()
		if err != nil {
			log.Warnf("Warn invalid address %s", addr)
			return
		}
		if strings.Contains(k, addr) {
			buf := new(bytes.Buffer)
			buf.Write(v)
			txh := common2.TransactionHistory{}
			txhd, _ := txh.Deserialize(buf)
			ret = append(ret, *txhd)
		}
	}

	return
}

func (m *MemPool) DeleteMemPoolTx(txid common.Uint256) {
	m.l.Lock()
	defer m.l.Unlock()
	for k := range m.p {
		if strings.Contains(k, txid.String()) {
			delete(m.p, k)
			delete(m.is_p, txid)
		}
	}

}
