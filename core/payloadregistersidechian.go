package core

import (
	"errors"
	"io"

	. "github.com/elastos/Elastos.ELA.Utility/common"
)

const RegisterSidechainPayloadVersion byte = 0x00

type SideChainType byte

const (
	Default    SideChainType = 0x00
	IDChain                  = 0x01
	TokenChain               = 0x02
)

type ConsensusType byte

const (
	PowConsensus ConsensusType = iota
	AuxPowConsensus
	PosConsensus
	DposConsensus
)

type ECType byte

const (
	Secp256r1 ECType = iota
	Secp256k1
)

type CheckPoint struct {
	Height    uint32
	Hash      Uint256
	Timestamp uint32
	Bits      uint32
}

func (cp *CheckPoint) Serialize(w io.Writer) error {
	if err := WriteUint32(w, cp.Height); err != nil {
		return errors.New("[CheckPoint], Height serialize failed.")
	}
	if err := cp.Hash.Serialize(w); err != nil {
		return errors.New("[CheckPoint], Hash serialize failed.")
	}
	if err := WriteUint32(w, cp.Timestamp); err != nil {
		return errors.New("[CheckPoint], Timestamp serialize failed.")
	}
	if err := WriteUint32(w, cp.Bits); err != nil {
		return errors.New("[CheckPoint], Bits serialize failed.")
	}
	return nil
}

func (cp *CheckPoint) Deserialize(r io.Reader) error {
	height, err := ReadUint32(r)
	if err != nil {
		return errors.New("[CheckPoint], Height deserialize failed.")
	}

	var hash Uint256
	if err = hash.Deserialize(r); err != nil {
		return errors.New("[CheckPoint], Hash deserialize failed.")
	}

	timestamp, err := ReadUint32(r)
	if err != nil {
		return errors.New("[CheckPoint], Timestamp deserialize failed.")
	}

	bits, err := ReadUint32(r)
	if err != nil {
		return errors.New("[CheckPoint], Bits deserialize failed.")
	}

	cp.Height = height
	cp.Hash = hash
	cp.Timestamp = timestamp
	cp.Bits = bits
	return nil
}

type PayloadRegisterSidechain struct {
	GenesisHash     Uint256
	CoinIndex       uint32
	Name            string
	SideChainType   SideChainType
	KnownPeers      []string
	CheckPoint      CheckPoint
	ConsensusType   ConsensusType
	BlockType       string
	TransactionType string
	ECType          ECType
	AddressType     string
	MinFee          Fixed64
	Rate            uint32
}

func (a *PayloadRegisterSidechain) Data(version byte) []byte {
	return []byte{0}
}

func (a *PayloadRegisterSidechain) Serialize(w io.Writer, version byte) error {
	if err := a.GenesisHash.Serialize(w); err != nil {
		return errors.New("[PayloadRegisterSidechain], GenesisHash serialize failed.")
	}
	if err := WriteUint32(w, a.CoinIndex); err != nil {
		return errors.New("[PayloadRegisterSidechain], CoinIndex serialize failed.")
	}
	if err := WriteVarString(w, a.Name); err != nil {
		return errors.New("[PayloadRegisterSidechain], Name serialize failed.")
	}
	if err := WriteUint8(w, uint8(a.SideChainType)); err != nil {
		return errors.New("[PayloadRegisterSidechain], SideChainType serialize failed.")
	}
	if err := WriteVarUint(w, uint64(len(a.KnownPeers))); err != nil {
		return errors.New("[PayloadRegisterSidechain], KnownPeers count serialize failed.")
	}
	for _, peer := range a.KnownPeers {
		if err := WriteVarString(w, peer); err != nil {
			return errors.New("[PayloadRegisterSidechain], KnownPeers serialize failed.")
		}
	}
	if err := a.CheckPoint.Serialize(w); err != nil {
		return err
	}
	if err := WriteUint8(w, uint8(a.ConsensusType)); err != nil {
		return errors.New("[PayloadRegisterSidechain], ConsensusType serialize failed.")
	}
	if err := WriteVarString(w, a.BlockType); err != nil {
		return errors.New("[PayloadRegisterSidechain], BlockType serialize failed.")
	}
	if err := WriteVarString(w, a.TransactionType); err != nil {
		return errors.New("[PayloadRegisterSidechain], TransactionType serialize failed.")
	}
	if err := WriteUint8(w, uint8(a.ECType)); err != nil {
		return errors.New("[PayloadRegisterSidechain], ECType serialize failed.")
	}
	if err := WriteVarString(w, a.AddressType); err != nil {
		return errors.New("[PayloadRegisterSidechain], AddressType serialize failed.")
	}
	if err := a.MinFee.Serialize(w); err != nil {
		return errors.New("[PayloadRegisterSidechain], MinFee serialize failed.")
	}
	if err := WriteUint32(w, a.Rate); err != nil {
		return errors.New("[PayloadRegisterSidechain], Rate serialize failed.")
	}

	return nil
}

func (a *PayloadRegisterSidechain) Deserialize(r io.Reader, version byte) error {
	var genesisHash Uint256
	if err := genesisHash.Deserialize(r); err != nil {
		return errors.New("[PayloadRegisterSidechain], GenesisHash deserialize failed.")
	}

	coinIndex, err := ReadUint32(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], CoinIndex deserialize failed.")
	}

	name, err := ReadVarString(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], Name deserialize failed.")
	}

	sideChainType, err := ReadUint8(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], SideChainType deserialize failed.")
	}

	count, err := ReadVarUint(r, 0)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], KnownPeers count deserialize failed")
	}

	knownPeers := make([]string, 0)
	for i := uint64(0); i < count; i++ {
		address, err := ReadVarString(r)
		if err != nil {
			return errors.New("[PayloadRegisterSidechain], KnownPeers deserialize failed.")
		}
		knownPeers = append(knownPeers, address)
	}

	var checkPoint CheckPoint
	if err := checkPoint.Deserialize(r); err != nil {
		return err
	}

	consensusType, err := ReadUint8(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], ConsensusType deserialize failed.")
	}

	blockType, err := ReadVarString(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], BlockType deserialize failed")
	}

	transactionType, err := ReadVarString(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], TransactionType deserialize failed")
	}

	ecType, err := ReadUint8(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], ECType deserialize failed.")
	}

	addressType, err := ReadVarString(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], AddressType deserialize failed")
	}

	var minFee Fixed64
	if err := minFee.Deserialize(r); err != nil {
		return errors.New("[PayloadRegisterSidechain], MinFee deserialize failed.")
	}

	rate, err := ReadUint32(r)
	if err != nil {
		return errors.New("[PayloadRegisterSidechain], Rate deserialize failed.")
	}

	a.GenesisHash = genesisHash
	a.CoinIndex = coinIndex
	a.Name = name
	a.SideChainType = SideChainType(sideChainType)
	a.KnownPeers = knownPeers
	a.CheckPoint = checkPoint
	a.ConsensusType = ConsensusType(consensusType)
	a.BlockType = blockType
	a.TransactionType = transactionType
	a.ECType = ECType(ecType)
	a.AddressType = addressType
	a.MinFee = minFee
	a.Rate = rate

	return nil
}
