package message

import (
	"Elastos.ELA/common/config"
	"Elastos.ELA/common/log"
	"Elastos.ELA/core/transaction"
	. "Elastos.ELA/errors"
	. "Elastos.ELA/net/protocol"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

// Transaction message
type txn struct {
	messageHeader
	txn transaction.Transaction
}

func (message txn) Handle(node Noder) error {
	log.Debug()
	log.Debug("RX Transaction message")
	tx := &message.txn
	if !node.LocalNode().ExistedID(tx.Hash()) {
		if errCode := node.LocalNode().AppendToTxnPool(&(message.txn)); errCode != Success {
			return errors.New("[message] VerifyTransaction failed when AppendToTxnPool.")
		}
		node.LocalNode().Relay(node, tx)
		log.Info("Relay Transaction")
		node.LocalNode().IncRxTxnCnt()
		log.Debug("RX Transaction message hash", message.txn.Hash())
		log.Debug("RX Transaction message type", message.txn.TxType)
	}

	return nil
}

func NewTxn(transaction *transaction.Transaction) ([]byte, error) {
	log.Debug()
	var msg txn

	msg.messageHeader.Magic = config.Parameters.Magic
	cmd := "tx"
	copy(msg.messageHeader.CMD[0:len(cmd)], cmd)
	tmpBuffer := bytes.NewBuffer([]byte{})
	transaction.Serialize(tmpBuffer)
	msg.txn = *transaction
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.LittleEndian, tmpBuffer.Bytes())
	if err != nil {
		log.Error("Binary Write failed at new Msg")
		return nil, err
	}
	s := sha256.Sum256(b.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.messageHeader.Checksum))
	msg.messageHeader.Length = uint32(len(b.Bytes()))
	log.Debug("The message payload length is ", msg.messageHeader.Length)

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func (message txn) Serialization() ([]byte, error) {
	headerBuffer, err := message.messageHeader.Serialization()
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(headerBuffer)
	message.txn.Serialize(buffer)

	return buffer.Bytes(), err
}

func (message *txn) Deserialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(message.messageHeader))
	err = message.txn.Deserialize(buf)
	if err != nil {
		return err
	}

	return nil
}
