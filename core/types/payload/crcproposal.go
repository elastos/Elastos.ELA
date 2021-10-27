// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package payload

import (
	"bytes"
	"errors"
	"io"
	"regexp"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/crypto"
)

const (
	// upgrade side chain proposal type
	MinUpgradeProposalType = 0x0200
	MaxUpgradeProposalType = 0x02ff
)

const (
	// Normal indicates the normal types of proposal.
	Normal CRCProposalType = 0x0000

	// 0x01 ELIP proposals.
	// ELIP indicates elastos improvement type of proposal.
	ELIP CRCProposalType = 0x0100
	// Used to identify process-related elips
	FLOWELIP CRCProposalType = 0x0101
	// Used to flag Elastos design issues
	INFOELIP CRCProposalType = 0x0102

	// 0x02 code upgrade related proposals.
	// MainChainUpgradeCode indicates the ELA code upgrade types of proposals.
	MainChainUpgradeCode CRCProposalType = 0x0200
	// DIDUpgradeCode indicates the DID code upgrade types of proposals.
	DIDUpgradeCode CRCProposalType = 0x0201
	// DIDUpgradeCode indicates the ETH code upgrade types of proposals.
	ETHUpgradeCode CRCProposalType = 0x0202

	// 0x03 for future usage.

	/// 0x04 main chain related proposals.
	// SecretaryGeneral indicates the vote secretary general types of proposals.
	SecretaryGeneral CRCProposalType = 0x0400
	// ChangeProposalOwner indicates the change proposal owner types of proposals.
	ChangeProposalOwner CRCProposalType = 0x0401
	// CloseProposal indicates the close proposal types of proposals.
	CloseProposal CRCProposalType = 0x0402
	// Registration of side chain.
	RegisterSideChain CRCProposalType = 0x0410

	// 0x04 DID related proposals.
	// Reserved did custom id.
	ReserveCustomID CRCProposalType = 0x0500
	// Receive did custom id.
	ReceiveCustomID CRCProposalType = 0x0501
	// The rate of custom id fee.
	ChangeCustomIDFee CRCProposalType = 0x0502
)

type CRCProposalType uint16

func (pt CRCProposalType) Name() string {
	switch pt {
	case Normal:
		return "Normal"
	case ELIP:
		return "ELIP"
		//todo if it is UpgradeCode should use if else
	//case MainChainUpgradeCode:
	//	return "MainChainUpgradeCode"
	//case DIDUpgradeCode:
	//	return "DIDUpgradeCode"
	//case ETHUpgradeCode:
	//	return "ETHUpgradeCode"
	case RegisterSideChain:
		return "RegisterSideChain"
	case ChangeProposalOwner:
		return "ChangeProposalOwner"
	case CloseProposal:
		return "CloseProposal"
	case SecretaryGeneral:
		return "SecretaryGeneral"
	case ReserveCustomID:
		return "ReserveCustomID"
	case ReceiveCustomID:
		return "ReceiveCustomID"
	case ChangeCustomIDFee:
		return "ChangeCustomIDFee"
	default:
		return "Unknown"
	}
}

const (
	// CRCProposalVersion indicates the version of CRC proposal payload
	CRCProposalVersion byte = 0x00
	//add draft data
	CRCProposalVersion01 byte = 0x01

	// MaxProposalDataSize the max size of proposal draft data or proposal
	// tracking document data.
	MaxProposalDataSize = 1 * 1024 * 1024

	CRCProposalUpgradeCodeVersion byte = 0x00
)

const (
	Imprest       InstallmentType = 0x00
	NormalPayment InstallmentType = 0x01
	FinalPayment  InstallmentType = 0x02
)

var SeedRegexp = regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})$`)

type InstallmentType byte

func (b InstallmentType) Name() string {
	switch b {
	case Imprest:
		return "Imprest"
	case NormalPayment:
		return "NormalPayment"
	case FinalPayment:
		return "FinalPayment"
	default:
		return "Unknown"
	}
}

type Budget struct {
	Type   InstallmentType
	Stage  byte
	Amount common.Fixed64
}

type CRCProposal struct {
	// The type of current CR Council proposal.
	ProposalType CRCProposalType

	// Used to store category data
	// with a length limit not exceeding 4096 characters
	CategoryData string

	// Public key of proposal owner.
	OwnerPublicKey []byte

	// The hash of draft proposal.
	DraftHash common.Uint256

	// Used to store draft data
	// with a length limit not exceeding 1M byte
	DraftData []byte

	// The detailed budget and expenditure plan.
	Budgets []Budget

	// The specified ELA address where the funds are to be sent.
	Recipient common.Uint168

	// Hash of proposal that need to change owner or need to be closed.
	TargetProposalHash common.Uint256

	// Reversed did custom id list.
	ReservedCustomIDList []string

	// Received did custom id list.
	ReceivedCustomIDList []string

	// Receiver did.
	ReceiverDID common.Uint168

	CustomIDFeeRateInfo

	// The specified ELA address where the funds are to be sent.
	NewRecipient common.Uint168

	// New public key of proposal owner.
	NewOwnerPublicKey []byte

	// Public key of SecretaryGeneral.
	SecretaryGeneralPublicKey []byte

	// DID of SecretaryGeneral.
	SecretaryGeneralDID common.Uint168

	// The signature of proposal's owner.
	Signature []byte

	// New proposal owner signature.
	NewOwnerSignature []byte

	// The signature of SecretaryGeneral.
	SecretaryGeneraSignature []byte

	// DID of CR Council Member.
	CRCouncilMemberDID common.Uint168

	// The signature of CR Council Member, check data include signature of
	// proposal owner.
	CRCouncilMemberSignature []byte

	//upgrade code info
	UpgradeCodeInfo *UpgradeCodeInfo

	// The registered side chain information
	SideChainInfo

	hash *common.Uint256
}

type UpgradeCodeInfo struct {
	//upgrade code working hegiht
	WorkingHeight uint32

	//node version
	NodeVersion string

	//node bin download url
	NodeDownLoadUrl string

	//node bin hash
	NodeBinHash *common.Uint256

	// if ForceUpgrade is true when height reaches WorkingHeight
	// version of msg.Version must greater or equal to NodeVersion
	ForceUpgrade bool
}

func (upgradeInfo *UpgradeCodeInfo) Serialize(w io.Writer, version byte) error {
	if err := common.WriteElement(w, upgradeInfo.WorkingHeight); err != nil {
		return errors.New("failed to serialize WorkingHeight")
	}
	if err := common.WriteVarString(w, upgradeInfo.NodeVersion); err != nil {
		return errors.New("failed to serialize NodeVersion")
	}
	if err := common.WriteVarString(w, upgradeInfo.NodeDownLoadUrl); err != nil {
		return errors.New("failed to serialize NodeDownLoadUrl")
	}
	if err := upgradeInfo.NodeBinHash.Serialize(w); err != nil {
		return errors.New("failed to serialize NodeBinHash")
	}

	if err := common.WriteElement(w, upgradeInfo.ForceUpgrade); err != nil {
		return errors.New("failed to serialize ForceUpgrade")
	}
	return nil
}

func (upgradeInfo *UpgradeCodeInfo) Deserialize(r io.Reader, version byte) error {
	var err error

	if err := common.ReadElement(r, &upgradeInfo.WorkingHeight); err != nil {
		return err
	}

	upgradeInfo.NodeVersion, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[UpgradeCodeInfo], NodeVersion deserialize failed")
	}

	upgradeInfo.NodeDownLoadUrl, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[UpgradeCodeInfo], NodeDownLoadUrl deserialize failed")
	}
	upgradeInfo.NodeBinHash = new(common.Uint256)
	if err = upgradeInfo.NodeBinHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize NodeBinHash")
	}

	if err := common.ReadElement(r, &upgradeInfo.ForceUpgrade); err != nil {
		return err
	}
	return nil
}

type SideChainInfo struct {
	// Name of side chain
	SideChainName string

	// Magic number of side chain
	MagicNumber uint32

	// Genesis hash of side chain
	GenesisHash common.Uint256

	// 1 ELA on main chain equals to how many coin on side chain
	ExchangeRate common.Fixed64

	// Effective height of register side chain
	EffectiveHeight uint32

	// Resource path
	ResourcePath string
}

func (sc *SideChainInfo) Serialize(w io.Writer) error {
	if err := common.WriteVarString(w, sc.SideChainName); err != nil {
		return errors.New("fail to serialize SideChainName")
	}
	if err := common.WriteUint32(w, sc.MagicNumber); err != nil {
		return errors.New("fail to serialize MagicNumber")
	}

	if err := sc.GenesisHash.Serialize(w); err != nil {
		return errors.New("failed to serialize GenesisHash")
	}

	if err := sc.ExchangeRate.Serialize(w); err != nil {
		return errors.New("failed to serialize ExchangeRate")
	}

	if err := common.WriteUint32(w, sc.EffectiveHeight); err != nil {
		return errors.New("failed to serialize EffectiveHeight")
	}

	if err := common.WriteVarString(w, sc.ResourcePath); err != nil {
		return errors.New("fail to serialize ResourcePath")
	}

	return nil
}

func (sc *SideChainInfo) Deserialize(r io.Reader) error {
	var err error
	sc.SideChainName, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], SideChainName deserialize failed")
	}

	sc.MagicNumber, err = common.ReadUint32(r)
	if err != nil {
		return errors.New("[CRCProposal], MagicNumber deserialize failed")
	}

	if err := sc.GenesisHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize GenesisHash")
	}

	err = sc.ExchangeRate.Deserialize(r)
	if err != nil {
		return errors.New("[CRCProposal], ExchangeRate deserialize failed")
	}

	sc.EffectiveHeight, err = common.ReadUint32(r)
	if err != nil {
		return errors.New("[CRCProposal], EffectiveHeight deserialize failed")
	}

	sc.ResourcePath, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], ResourcePath deserialize failed")
	}
	return nil
}

type CustomIDFeeRateInfo struct {
	// The rate of custom DID fee.
	RateOfCustomIDFee common.Fixed64

	// Effective at the side chain height of EID.
	EIDEffectiveHeight uint32
}

func (sc *CustomIDFeeRateInfo) Serialize(w io.Writer) error {
	if err := sc.RateOfCustomIDFee.Serialize(w); err != nil {
		return errors.New("failed to serialize RateOfCustomIDFee")
	}

	if err := common.WriteUint32(w, sc.EIDEffectiveHeight); err != nil {
		return errors.New("failed to serialize EIDEffectiveHeight")
	}

	return nil
}

func (sc *CustomIDFeeRateInfo) Deserialize(r io.Reader) error {
	var err error
	if err = sc.RateOfCustomIDFee.Deserialize(r); err != nil {
		return errors.New("failed to deserialize RateOfCustomIDFee")
	}

	sc.EIDEffectiveHeight, err = common.ReadUint32(r)
	if err != nil {
		return errors.New("failed to deserialize EIDEffectiveHeight")
	}
	return nil
}

func (p *CRCProposal) Data(version byte) []byte {
	buf := new(bytes.Buffer)
	if err := p.SerializeUnsigned(buf, version); err != nil {
		return []byte{0}
	}

	return buf.Bytes()
}

func (p *CRCProposal) SerializeUnsigned(w io.Writer, version byte) error {
	switch p.ProposalType {
	case ChangeProposalOwner:
		return p.SerializeUnsignedChangeProposalOwner(w, version)
	case CloseProposal:
		return p.SerializeUnsignedCloseProposal(w, version)
	case SecretaryGeneral:
		return p.SerializeUnsignedChangeSecretaryGeneral(w, version)
	case MainChainUpgradeCode, DIDUpgradeCode, ETHUpgradeCode:
		return p.SerializeUnsignedUpgradeCode(w, version)
	case ReserveCustomID:
		return p.SerializeUnsignedReservedCustomID(w, version)
	case ReceiveCustomID:
		return p.SerializeUnsignedReceivedCustomID(w, version)
	case ChangeCustomIDFee:
		return p.SerializeUnsignedChangeCustomIDFee(w, version)
	case RegisterSideChain:
		return p.SerializeUnsignedRegisterSideChain(w, version)
	default:
		return p.SerializeUnsignedNormalOrELIP(w, version)
	}
	return nil
}

func (p *CRCProposal) SerializeUnsignedNormalOrELIP(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}

	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}

	if err := common.WriteVarUint(w, uint64(len(p.Budgets))); err != nil {
		return errors.New("failed to serialize Budgets")
	}

	for _, v := range p.Budgets {
		if err := v.Serialize(w); err != nil {
			return errors.New("failed to serialize Budgets")
		}
	}

	if err := p.Recipient.Serialize(w); err != nil {
		return errors.New("failed to serialize Recipient")
	}

	return nil
}

func (p *CRCProposal) SerializeUnsignedChangeProposalOwner(w io.Writer, version byte) error {
	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}
	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}
	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}
	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}
	if err := p.TargetProposalHash.Serialize(w); err != nil {
		return errors.New("failed to serialize TargetProposalHash")
	}
	if err := p.NewRecipient.Serialize(w); err != nil {
		return errors.New("failed to serialize Recipient")
	}
	if err := common.WriteVarBytes(w, p.NewOwnerPublicKey); err != nil {
		return errors.New("failed to serialize NewOwnerPublicKey")
	}
	return nil
}

func (p *CRCProposal) SerializeUnsignedChangeSecretaryGeneral(w io.Writer, version byte) error {
	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}
	if err := common.WriteVarBytes(w, p.SecretaryGeneralPublicKey); err != nil {
		return errors.New("failed to serialize SecretaryGeneralPublicKey")
	}

	if err := p.SecretaryGeneralDID.Serialize(w); err != nil {
		return errors.New("failed to serialize SecretaryGeneralDID")
	}
	return nil
}

func (p *CRCProposal) SerializeUnsignedUpgradeCode(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	//UpgradeCodeInfo Serialize
	if p.UpgradeCodeInfo != nil {
		if err := p.UpgradeCodeInfo.Serialize(w, CRCProposalUpgradeCodeVersion); err != nil {
			return errors.New("failed to serialize UpgradeCodeInfo")
		}
	}
	return nil
}

func (p *CRCProposal) SerializeUnsignedRegisterSideChain(w io.Writer, version byte) error {
	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("[CRCProposal], Category Data serialize failed")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}

	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}

	if err := p.SideChainInfo.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (p *CRCProposal) SerializeUnsignedCloseProposal(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}
	if err := p.TargetProposalHash.Serialize(w); err != nil {
		return errors.New("failed to serialize CloseProposalHash")
	}

	return nil
}

func (p *CRCProposal) SerializeUnsignedChangeCustomIDFee(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}

	if err := p.CustomIDFeeRateInfo.Serialize(w); err != nil {
		return errors.New("failed to serialize CustomIDFeeRateInfo")
	}

	return nil
}

func (p *CRCProposal) SerializeUnsignedReceivedCustomID(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}

	if err := common.WriteVarUint(w, uint64(len(p.ReceivedCustomIDList))); err != nil {
		return errors.New("failed to serialize ReceivedCustomIDList len")
	}

	for _, v := range p.ReceivedCustomIDList {
		if err := common.WriteVarString(w, v); err != nil {
			return errors.New("failed to serialize ReceivedCustomIDList")
		}
	}

	if err := p.ReceiverDID.Serialize(w); err != nil {
		return errors.New("failed to serialize ReceiverDID")
	}

	return nil
}

func (p *CRCProposal) SerializeUnsignedReservedCustomID(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("[CRCProposal], Category Data serialize failed")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		if err := common.WriteVarBytes(w, p.DraftData); err != nil {
			return errors.New("failed to serialize DraftData")
		}
	}

	if err := common.WriteVarUint(w, uint64(len(p.ReservedCustomIDList))); err != nil {
		return errors.New("failed to serialize ReservedCustomIDList len")
	}

	for _, v := range p.ReservedCustomIDList {
		if err := common.WriteVarString(w, v); err != nil {
			return errors.New("failed to serialize ReservedCustomIDList")
		}
	}

	return nil
}

func (p *CRCProposal) Serialize(w io.Writer, version byte) error {
	switch p.ProposalType {
	case ChangeProposalOwner:
		return p.SerializeChangeProposalOwner(w, version)
	case CloseProposal:
		return p.SerializeCloseProposal(w, version)
	case SecretaryGeneral:
		return p.SerializeChangeSecretaryGeneral(w, version)
	case MainChainUpgradeCode, DIDUpgradeCode, ETHUpgradeCode:
		return p.SerializeUpgradeCode(w, version)
	case RegisterSideChain:
		return p.SerializeRegisterSideChain(w, version)
	default:
		return p.SerializeNormalOrELIP(w, version)
	}
}

func (p *CRCProposal) SerializeNormalOrELIP(w io.Writer, version byte) error {
	if err := p.SerializeUnsigned(w, version); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return common.WriteVarBytes(w, p.CRCouncilMemberSignature)
}

func (p *CRCProposal) SerializeChangeProposalOwner(w io.Writer, version byte) error {
	if err := p.SerializeUnsigned(w, version); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return errors.New("failed to serialize Signature")
	}
	if err := common.WriteVarBytes(w, p.NewOwnerSignature); err != nil {
		return errors.New("failed to serialize NewOwnerSignature")
	}
	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}
	return common.WriteVarBytes(w, p.CRCouncilMemberSignature)
}

func (p *CRCProposal) SerializeUpgradeCode(w io.Writer, version byte) error {
	if err := p.SerializeUnsigned(w, version); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return common.WriteVarBytes(w, p.CRCouncilMemberSignature)
}

func (p *CRCProposal) SerializeChangeSecretaryGeneral(w io.Writer, version byte) error {

	if err := p.SerializeUnsignedChangeSecretaryGeneral(w, version); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, p.SecretaryGeneraSignature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return common.WriteVarBytes(w, p.CRCouncilMemberSignature)
}

func (p *CRCProposal) SerializeRegisterSideChain(w io.Writer, version byte) error {
	if err := p.SerializeUnsigned(w, version); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return common.WriteVarBytes(w, p.CRCouncilMemberSignature)
}

func (p *CRCProposal) SerializeCloseProposal(w io.Writer, version byte) error {
	if err := p.SerializeUnsigned(w, version); err != nil {
		return err
	}

	if err := common.WriteVarBytes(w, p.Signature); err != nil {
		return err
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	return common.WriteVarBytes(w, p.CRCouncilMemberSignature)
}

func (b *Budget) Serialize(w io.Writer) error {
	if err := common.WriteElement(w, b.Type); err != nil {
		return errors.New("failed to serialize Type")
	}
	if err := common.WriteElement(w, b.Stage); err != nil {
		return errors.New("failed to serialize Stage")
	}
	return b.Amount.Serialize(w)
}

func (b *Budget) Deserialize(r io.Reader) error {
	if err := common.ReadElement(r, &b.Type); err != nil {
		return errors.New("[CRCProposal], Type deserialize failed")
	}
	if err := common.ReadElement(r, &b.Stage); err != nil {
		return errors.New("[CRCProposal], Stage deserialize failed")
	}
	return b.Amount.Deserialize(r)

}

func (p *CRCProposal) DeserializeUnSigned(r io.Reader, version byte) error {
	switch p.ProposalType {
	case ChangeProposalOwner:
		return p.DeserializeUnSignedChangeProposalOwner(r, version)
	case CloseProposal:
		return p.DeserializeUnSignedCloseProposal(r, version)
	case SecretaryGeneral:
		return p.DeserializeUnSignedChangeSecretaryGeneral(r, version)
	case MainChainUpgradeCode, DIDUpgradeCode, ETHUpgradeCode:
		return p.DeserializeUnsignedUpgradeCode(r, version)
	case ReserveCustomID:
		return p.DeserializeUnSignedReservedCustomID(r, version)
	case ReceiveCustomID:
		return p.DeserializeUnSignedReceivedCustomID(r, version)
	case ChangeCustomIDFee:
		return p.DeserializeUnSignedChangeCustomIDFee(r, version)
	case RegisterSideChain:
		return p.DeserializeUnsignedRegisterSideChain(r, version)
	default:
		return p.DeserializeUnSignedNormalOrELIP(r, version)
	}
}

func (p *CRCProposal) DeserializeUnsignedUpgradeCode(r io.Reader, version byte) error {
	var err error
	if p.CategoryData, err = common.ReadVarString(r); err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}
	if p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner"); err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}
	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	//
	p.UpgradeCodeInfo = new(UpgradeCodeInfo)
	//UpgradeCodeInfo Deserialize
	if err := p.UpgradeCodeInfo.Deserialize(r, CRCProposalUpgradeCodeVersion); err != nil {
		return errors.New("failed to serialize UpgradeCodeInfo")
	}
	return nil
}

func (p *CRCProposal) DeserializeUnSignedNormalOrELIP(r io.Reader, version byte) error {
	var err error
	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}
	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return errors.New("failed to deserialize Budgets")
	}
	p.Budgets = make([]Budget, 0)
	for i := 0; i < int(count); i++ {
		var budget Budget
		if err := budget.Deserialize(r); err != nil {
			return errors.New("failed to deserialize Budgets")
		}
		p.Budgets = append(p.Budgets, budget)
	}

	if err = p.Recipient.Deserialize(r); err != nil {
		return errors.New("failed to deserialize Recipient")
	}

	return nil
}

func (p *CRCProposal) DeserializeUnSignedChangeProposalOwner(r io.Reader, version byte) error {
	var err error
	if p.CategoryData, err = common.ReadVarString(r); err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}
	if p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner"); err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}
	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}
	if err = p.TargetProposalHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize TargetProposalHash")
	}
	if err = p.NewRecipient.Deserialize(r); err != nil {
		return errors.New("failed to deserialize Recipient")
	}
	if p.NewOwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner"); err != nil {
		return errors.New("failed to deserialize NewOwnerPublicKey")
	}
	return nil
}

func (p *CRCProposal) DeserializeUnsignedRegisterSideChain(r io.Reader, version byte) error {
	var err error

	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}

	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}

	if err = p.SideChainInfo.Deserialize(r); err != nil {
		return err
	}

	return nil
}

func (p *CRCProposal) DeserializeUnSignedCloseProposal(r io.Reader, version byte) error {
	var err error

	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}
	if err = p.TargetProposalHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CloseProposalHash")
	}

	return nil
}

func (p *CRCProposal) DeserializeUnSignedChangeCustomIDFee(r io.Reader, version byte) error {
	var err error

	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}

	if err = p.CustomIDFeeRateInfo.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CustomIDFeeRateInfo")
	}

	return nil
}

func (p *CRCProposal) DeserializeUnSignedReceivedCustomID(r io.Reader, version byte) error {
	var err error

	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return errors.New("failed to deserialize Budgets")
	}
	p.ReceivedCustomIDList = make([]string, 0)
	for i := 0; i < int(count); i++ {
		customID, err := common.ReadVarString(r)
		if err != nil {
			return errors.New("[CRCProposal], reserved custom id deserialize failed")
		}

		p.ReceivedCustomIDList = append(p.ReceivedCustomIDList, customID)
	}

	if err = p.ReceiverDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize ReceiverDID")
	}

	return nil
}

func (p *CRCProposal) DeserializeUnSignedReservedCustomID(r io.Reader, version byte) error {
	var err error

	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return errors.New("failed to deserialize Budgets")
	}
	p.ReservedCustomIDList = make([]string, 0)
	for i := 0; i < int(count); i++ {
		customID, err := common.ReadVarString(r)
		if err != nil {
			return errors.New("[CRCProposal], reserved custom id deserialize failed")
		}

		p.ReservedCustomIDList = append(p.ReservedCustomIDList, customID)
	}

	return nil
}

func (p *CRCProposal) DeserializeUnSignedChangeSecretaryGeneral(r io.Reader, version byte) error {
	var err error
	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("[CRCProposal], Category data deserialize failed")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}
	if version >= CRCProposalVersion01 {
		p.DraftData, err = common.ReadVarBytes(r, MaxProposalDataSize, "draft data")
		if err != nil {
			return errors.New("failed to deserialize draft data")
		}
	}
	p.SecretaryGeneralPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "secretarygeneralpublickey")
	if err != nil {
		return errors.New("failed to deserialize SecretaryGeneralPublicKey")
	}
	if err := p.SecretaryGeneralDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize SecretaryGeneralDID")
	}
	return nil
}

func (p *CRCProposal) Deserialize(r io.Reader, version byte) error {
	err := common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("[CRCProposal], ProposalType deserialize failed")
	}
	switch p.ProposalType {
	case ChangeProposalOwner:
		return p.DeserializeChangeProposalOwner(r, version)
	case CloseProposal:
		return p.DeserializeCloseProposal(r, version)
	case SecretaryGeneral:
		return p.DeserializeChangeSecretaryGeneral(r, version)
	case MainChainUpgradeCode, DIDUpgradeCode, ETHUpgradeCode:
		return p.DeserializeUpgradeCode(r, version)
	case RegisterSideChain:
		return p.DeserializeRegisterSideChain(r, version)
	default:
		return p.DeserializeNormalOrELIP(r, version)
	}
}

func (p *CRCProposal) DeserializeNormalOrELIP(r io.Reader, version byte) error {
	if err := p.DeserializeUnSigned(r, version); err != nil {
		return err
	}

	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	CRCouncilMemberSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "CR sign data")
	if err != nil {
		return err
	}
	p.CRCouncilMemberSignature = CRCouncilMemberSignature

	return nil
}

func (p *CRCProposal) DeserializeUpgradeCode(r io.Reader, version byte) error {
	if err := p.DeserializeUnsignedUpgradeCode(r, version); err != nil {
		return err
	}

	// owner signature
	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}
	// cr signature
	CRCouncilMemberSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "CR sign data")
	if err != nil {
		return err
	}
	p.CRCouncilMemberSignature = CRCouncilMemberSignature
	return nil
}

func (p *CRCProposal) DeserializeChangeProposalOwner(r io.Reader, version byte) error {
	if err := p.DeserializeUnSigned(r, version); err != nil {
		return err
	}

	// owner signature
	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	// new owner signature
	newOwnerSign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.NewOwnerSignature = newOwnerSign

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}
	// cr signature
	CRCouncilMemberSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "CR sign data")
	if err != nil {
		return err
	}
	p.CRCouncilMemberSignature = CRCouncilMemberSignature
	return nil
}

func (p *CRCProposal) DeserializeRegisterSideChain(r io.Reader, version byte) error {

	if err := p.DeserializeUnSigned(r, version); err != nil {
		return err
	}
	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign
	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	CRCouncilMemberSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "CR sign data")
	if err != nil {
		return err
	}
	p.CRCouncilMemberSignature = CRCouncilMemberSignature

	return nil
}

func (p *CRCProposal) DeserializeCloseProposal(r io.Reader, version byte) error {

	if err := p.DeserializeUnSigned(r, version); err != nil {
		return err
	}

	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	CRCouncilMemberSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "CR sign data")
	if err != nil {
		return err
	}
	p.CRCouncilMemberSignature = CRCouncilMemberSignature

	return nil
}

func (p *CRCProposal) DeserializeChangeSecretaryGeneral(r io.Reader, version byte) error {

	if err := p.DeserializeUnSignedChangeSecretaryGeneral(r, version); err != nil {
		return err
	}

	sign, err := common.ReadVarBytes(r, crypto.SignatureLength, "sign data")
	if err != nil {
		return err
	}
	p.Signature = sign

	SecretaryGeneraSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "secretary general sign data")
	if err != nil {
		return err
	}
	p.SecretaryGeneraSignature = SecretaryGeneraSignature

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	CRCouncilMemberSignature, err := common.ReadVarBytes(r, crypto.SignatureLength, "CR sign data")
	if err != nil {
		return err
	}
	p.CRCouncilMemberSignature = CRCouncilMemberSignature
	return nil
}

func (p *CRCProposal) Hash(payloadVersion byte) common.Uint256 {
	if p.hash == nil {
		buf := new(bytes.Buffer)
		p.Serialize(buf, payloadVersion)
		hash := common.Hash(buf.Bytes())
		p.hash = &hash
	}
	return *p.hash
}

func (p *CRCProposal) ToProposalInfo(payloadVersion byte) CRCProposalInfo {
	info := CRCProposalInfo{
		ProposalType:              p.ProposalType,
		CategoryData:              p.CategoryData,
		OwnerPublicKey:            p.OwnerPublicKey,
		DraftHash:                 p.DraftHash,
		Budgets:                   p.Budgets,
		Recipient:                 p.Recipient,
		TargetProposalHash:        p.TargetProposalHash,
		ReservedCustomIDList:      p.ReservedCustomIDList,
		ReceivedCustomIDList:      p.ReceivedCustomIDList,
		ReceiverDID:               p.ReceiverDID,
		RateOfCustomIDFee:         p.RateOfCustomIDFee,
		EIDEffectiveHeight:        p.EIDEffectiveHeight,
		NewRecipient:              p.NewRecipient,
		NewOwnerPublicKey:         p.NewOwnerPublicKey,
		SecretaryGeneralPublicKey: p.SecretaryGeneralPublicKey,
		SecretaryGeneralDID:       p.SecretaryGeneralDID,
		CRCouncilMemberDID:        p.CRCouncilMemberDID,
		SideChainInfo:             p.SideChainInfo,
		Hash:                      p.Hash(payloadVersion),
	}

	if info.Budgets == nil {
		info.Budgets = []Budget{}
	}
	if info.ReservedCustomIDList == nil {
		info.ReservedCustomIDList = []string{}
	}
	if info.ReceivedCustomIDList == nil {
		info.ReceivedCustomIDList = []string{}
	}
	if info.NewOwnerPublicKey == nil {
		info.NewOwnerPublicKey = []byte{}
	}
	if info.SecretaryGeneralPublicKey == nil {
		info.SecretaryGeneralPublicKey = []byte{}
	}

	return info
}

type CRCProposalInfo struct {
	// The type of current CR Council proposal.
	ProposalType CRCProposalType

	// Used to store category data
	// with a length limit not exceeding 4096 characters
	CategoryData string

	// Public key of proposal owner.
	OwnerPublicKey []byte

	// The hash of draft proposal.
	DraftHash common.Uint256

	// The detailed budget and expenditure plan.
	Budgets []Budget

	// The specified ELA address where the funds are to be sent.
	Recipient common.Uint168

	// Hash of proposal that need to change owner or need to be closed.
	TargetProposalHash common.Uint256

	// Reversed did custom id list.
	ReservedCustomIDList []string

	// Received did custom id list.
	ReceivedCustomIDList []string

	// Receiver did.
	ReceiverDID common.Uint168

	// The rate of custom DID fee.
	RateOfCustomIDFee common.Fixed64

	// The effective height of EID.
	EIDEffectiveHeight uint32

	// The specified ELA address where the funds are to be sent.
	NewRecipient common.Uint168

	// New public key of proposal owner.
	NewOwnerPublicKey []byte

	// Public key of SecretaryGeneral.
	SecretaryGeneralPublicKey []byte

	// DID of SecretaryGeneral.
	SecretaryGeneralDID common.Uint168

	// DID of CR Council Member.
	CRCouncilMemberDID common.Uint168

	SideChainInfo

	// The proposal hash
	Hash common.Uint256
}

// only used to save into check point.
func (p *CRCProposalInfo) Serialize(w io.Writer, version byte) error {

	if err := common.WriteElement(w, p.ProposalType); err != nil {
		return errors.New("failed to serialize ProposalType")
	}

	if err := common.WriteVarString(w, p.CategoryData); err != nil {
		return errors.New("failed to serialize CategoryData")
	}

	if err := common.WriteVarBytes(w, p.OwnerPublicKey); err != nil {
		return errors.New("failed to serialize OwnerPublicKey")
	}

	if err := p.DraftHash.Serialize(w); err != nil {
		return errors.New("failed to serialize DraftHash")
	}

	if err := common.WriteVarUint(w, uint64(len(p.Budgets))); err != nil {
		return errors.New("failed to serialize Budgets")
	}

	for _, v := range p.Budgets {
		if err := v.Serialize(w); err != nil {
			return errors.New("failed to serialize Budgets")
		}
	}

	if err := p.Recipient.Serialize(w); err != nil {
		return errors.New("failed to serialize Recipient")
	}

	if err := p.TargetProposalHash.Serialize(w); err != nil {
		return errors.New("failed to serialize TargetProposalHash")
	}

	if err := common.WriteVarUint(w, uint64(len(p.ReservedCustomIDList))); err != nil {
		return errors.New("failed to serialize ReservedCustomIDList len")
	}

	for _, v := range p.ReservedCustomIDList {
		if err := common.WriteVarString(w, v); err != nil {
			return errors.New("failed to serialize ReservedCustomIDList")
		}
	}

	if err := common.WriteVarUint(w, uint64(len(p.ReceivedCustomIDList))); err != nil {
		return errors.New("failed to serialize ReceivedCustomIDList len")
	}

	for _, v := range p.ReceivedCustomIDList {
		if err := common.WriteVarString(w, v); err != nil {
			return errors.New("failed to serialize ReceivedCustomIDList")
		}
	}

	if err := p.ReceiverDID.Serialize(w); err != nil {
		return errors.New("failed to serialize ReceiverDID")
	}

	if err := p.RateOfCustomIDFee.Serialize(w); err != nil {
		return errors.New("failed to serialize RateOfCustomIDFee")
	}

	if err := common.WriteUint32(w, p.EIDEffectiveHeight); err != nil {
		return errors.New("failed to serialize EIDEffectiveHeight")
	}

	if err := p.NewRecipient.Serialize(w); err != nil {
		return errors.New("failed to serialize Recipient")
	}

	if err := common.WriteVarBytes(w, p.NewOwnerPublicKey); err != nil {
		return errors.New("failed to serialize NewOwnerPublicKey")
	}

	if err := common.WriteVarBytes(w, p.SecretaryGeneralPublicKey); err != nil {
		return errors.New("failed to serialize SecretaryGeneralPublicKey")
	}

	if err := p.SecretaryGeneralDID.Serialize(w); err != nil {
		return errors.New("failed to serialize SecretaryGeneralDID")
	}

	if err := p.CRCouncilMemberDID.Serialize(w); err != nil {
		return errors.New("failed to serialize CRCouncilMemberDID")
	}

	if err := p.SideChainInfo.Serialize(w); err != nil {
		return errors.New("failed to serialize SideChainInfo")
	}

	if err := p.Hash.Serialize(w); err != nil {
		return errors.New("failed to serialize Hash")
	}
	return nil
}

// only used to save into check point.
func (p *CRCProposalInfo) Deserialize(r io.Reader, version byte) error {
	err := common.ReadElement(r, &p.ProposalType)
	if err != nil {
		return errors.New("failed to deserialize ProposalType")
	}

	p.CategoryData, err = common.ReadVarString(r)
	if err != nil {
		return errors.New("failed to deserialize CategoryData")
	}

	p.OwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner")
	if err != nil {
		return errors.New("failed to deserialize OwnerPublicKey")
	}

	if err = p.DraftHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize DraftHash")
	}

	var count uint64
	if count, err = common.ReadVarUint(r, 0); err != nil {
		return errors.New("failed to deserialize Budgets")
	}
	p.Budgets = make([]Budget, 0)
	for i := 0; i < int(count); i++ {
		var budget Budget
		if err := budget.Deserialize(r); err != nil {
			return errors.New("failed to deserialize Budgets")
		}
		p.Budgets = append(p.Budgets, budget)
	}

	if err = p.Recipient.Deserialize(r); err != nil {
		return errors.New("failed to deserialize Recipient")
	}

	if err = p.TargetProposalHash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize TargetProposalHash")
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return errors.New("failed to deserialize Budgets")
	}
	p.ReservedCustomIDList = make([]string, 0)
	for i := 0; i < int(count); i++ {
		customID, err := common.ReadVarString(r)
		if err != nil {
			return errors.New("[CRCProposal], reserved custom id deserialize failed")
		}

		p.ReservedCustomIDList = append(p.ReservedCustomIDList, customID)
	}

	if count, err = common.ReadVarUint(r, 0); err != nil {
		return errors.New("failed to deserialize Budgets")
	}
	p.ReceivedCustomIDList = make([]string, 0)
	for i := 0; i < int(count); i++ {
		customID, err := common.ReadVarString(r)
		if err != nil {
			return errors.New("[CRCProposal], reserved custom id deserialize failed")
		}

		p.ReceivedCustomIDList = append(p.ReceivedCustomIDList, customID)
	}

	if err = p.ReceiverDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize ReceiverDID")
	}

	if err = p.RateOfCustomIDFee.Deserialize(r); err != nil {
		return errors.New("failed to deserialize RateOfCustomIDFee")
	}

	if p.EIDEffectiveHeight, err = common.ReadUint32(r); err != nil {
		return errors.New("failed to deserialize EIDEffectiveHeight")
	}

	if err = p.NewRecipient.Deserialize(r); err != nil {
		return errors.New("failed to deserialize Recipient")
	}

	if p.NewOwnerPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "owner"); err != nil {
		return errors.New("failed to deserialize NewOwnerPublicKey")
	}

	p.SecretaryGeneralPublicKey, err = common.ReadVarBytes(r, crypto.NegativeBigLength, "secretarygeneralpublickey")
	if err != nil {
		return errors.New("failed to deserialize SecretaryGeneralPublicKey")
	}

	if err := p.SecretaryGeneralDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize SecretaryGeneralDID")
	}

	if err := p.CRCouncilMemberDID.Deserialize(r); err != nil {
		return errors.New("failed to deserialize CRCouncilMemberDID")
	}

	if err := p.SideChainInfo.Deserialize(r); err != nil {
		return errors.New("failed to deserialize SideChainInfo")
	}

	if err := p.Hash.Deserialize(r); err != nil {
		return errors.New("failed to deserialize Hash")
	}
	return nil
}
