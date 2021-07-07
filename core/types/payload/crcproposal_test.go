package payload

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/contract"
	"github.com/elastos/Elastos.ELA/crypto"
	"github.com/stretchr/testify/assert"
)

func TestCRCProposal_Serialize(t *testing.T) {

	crcProposalPayload := &CRCProposal{
		ProposalType:       MainChainUpgradeCode,
		CategoryData:       "CategoryData",
		OwnerPublicKey:     []byte{},
		DraftHash:          common.Uint256{},
		CRCouncilMemberDID: common.Uint168{},
		UpgradeCodeInfo: &UpgradeCodeInfo{
			WorkingHeight:   123,
			NodeVersion:     "0.1.2",
			NodeDownLoadUrl: "https://www.google.com/",
			NodeBinHash:     common.Uint256{},
			ForceUpgrade:    false,
		},
	}
	buf := new(bytes.Buffer)
	crcProposalPayload.Serialize(buf, CRCProposalVersion)

	crPayload2 := &CRCProposal{}
	crPayload2.Deserialize(buf, CRCProposalVersion)

	fmt.Println("crPayload2", crPayload2)
	result := crPayload2.IsEqual(crcProposalPayload, CRCProposalVersion)
	assert.True(t, result)
}

func TestCRCProposal_SerializeUpgradeCode(t *testing.T) {
	ownerPublicKeyStr1 := "02f981e4dae4983a5d284d01609ad735e3242c5672bb2c7bb0018cc36f9ab0c4a5"
	ownerPrivateKeyStr1 := "15e0947580575a9b6729570bed6360a890f84a07dc837922fe92275feec837d4"
	crPublicKeyStr := "036db5984e709d2e0ec62fd974283e9a18e7b87e8403cc784baf1f61f775926535"
	crPrivateKeyStr := "b2c25e877c8a87d54e8a20a902d27c7f24ed52810813ba175ca4e8d3036d130e"

	ownerPublicKey, _ := common.HexStringToBytes(ownerPublicKeyStr1)
	ownerPrivateKey, _ := common.HexStringToBytes(ownerPrivateKeyStr1)
	crPrivateKey, _ := common.HexStringToBytes(crPrivateKeyStr)
	crCode := getCodeByPubKeyStr(crPublicKeyStr)
	crDID, _ := getDIDFromCode(crCode)

	type fields struct {
		ProposalType              CRCProposalType
		CategoryData              string
		OwnerPublicKey            []byte
		DraftHash                 common.Uint256
		DraftData                 []byte
		Budgets                   []Budget
		Recipient                 common.Uint168
		TargetProposalHash        common.Uint256
		ReservedCustomIDList      []string
		ReceivedCustomIDList      []string
		ReceiverDID               common.Uint168
		RateOfCustomIDFee         common.Fixed64
		NewRecipient              common.Uint168
		NewOwnerPublicKey         []byte
		SecretaryGeneralPublicKey []byte
		SecretaryGeneralDID       common.Uint168
		Signature                 []byte
		NewOwnerSignature         []byte
		SecretaryGeneraSignature  []byte
		CRCouncilMemberDID        common.Uint168
		CRCouncilMemberSignature  []byte
		UpgradeCodeInfo           *UpgradeCodeInfo
		hash                      *common.Uint256
	}
	type args struct {
		version byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "TestCRCProposal_SerializeUpgradeCode",
			fields: fields{
				ProposalType:       MainChainUpgradeCode,
				CategoryData:       "CategoryData",
				OwnerPublicKey:     ownerPublicKey,
				DraftHash:          common.Uint256{},
				CRCouncilMemberDID: *crDID,
				UpgradeCodeInfo: &UpgradeCodeInfo{
					WorkingHeight:   14,
					NodeVersion:     "123",
					NodeDownLoadUrl: "https://www.google.com/",
					NodeBinHash:     common.Uint256{},
					ForceUpgrade:    false,
				},
			},
			args:    args{version: 0},
			wantErr: false,
			wantW:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crcProposalPayload := &CRCProposal{
				ProposalType:             tt.fields.ProposalType,
				CategoryData:             tt.fields.CategoryData,
				OwnerPublicKey:           tt.fields.OwnerPublicKey,
				DraftHash:                tt.fields.DraftHash,
				Signature:                tt.fields.Signature,
				CRCouncilMemberDID:       tt.fields.CRCouncilMemberDID,
				CRCouncilMemberSignature: tt.fields.CRCouncilMemberSignature,
				UpgradeCodeInfo:          tt.fields.UpgradeCodeInfo,
			}
			signBuf := new(bytes.Buffer)
			crcProposalPayload.SerializeUnsigned(signBuf, tt.args.version)
			sig, _ := crypto.Sign(ownerPrivateKey, signBuf.Bytes())
			crcProposalPayload.Signature = sig

			crcProposalPayload.CRCouncilMemberDID.Serialize(signBuf)
			crSig, _ := crypto.Sign(crPrivateKey, signBuf.Bytes())
			crcProposalPayload.CRCouncilMemberSignature = crSig

			w := &bytes.Buffer{}
			err := crcProposalPayload.SerializeUpgradeCode(w, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("SerializeUpgradeCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			crcProposalPayload2 := &CRCProposal{}
			err = common.ReadElement(w, &crcProposalPayload2.ProposalType)
			if err != nil {
				fmt.Println(err)
			}
			crcProposalPayload2.DeserializeUpgradeCode(w, tt.args.version)
			result := crcProposalPayload2.IsEqual(crcProposalPayload, tt.args.version)
			assert.True(t, result)
		})
	}
}

func getDIDFromCode(code []byte) (*common.Uint168, error) {
	newCode := make([]byte, len(code))
	copy(newCode, code)
	didCode := append(newCode[:len(newCode)-1], common.DID)

	if ct1, err := contract.CreateCRIDContractByCode(didCode); err != nil {
		return nil, err
	} else {
		return ct1.ToProgramHash(), nil
	}
}

func getCodeByPubKeyStr(publicKey string) []byte {
	pkBytes, _ := common.HexStringToBytes(publicKey)
	pk, _ := crypto.DecodePoint(pkBytes)
	redeemScript, _ := contract.CreateStandardRedeemScript(pk)
	return redeemScript
}

func TestIsUpgradeCodeProposal(t *testing.T) {
	type args struct {
		ProposalType CRCProposalType
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
		{
			name: "MainChainUpgradeCode",
			args: args{
				ProposalType: MainChainUpgradeCode,
			},
			want: true,
		},
		{
			name: "DIDUpgradeCode",
			args: args{
				ProposalType: DIDUpgradeCode,
			},
			want: true,
		},
		{
			name: "ETHUpgradeCode",
			args: args{
				ProposalType: ETHUpgradeCode,
			},
			want: true,
		},
		{
			name: "0x02ff",
			args: args{
				ProposalType: 0x02ff,
			},
			want: true,
		},
		{
			name: "0xff",
			args: args{
				ProposalType: 0xff,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUpgradeCodeProposal(tt.args.ProposalType); got != tt.want {
				t.Errorf("IsUpgradeCodeProposal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpgradeCodeInfo_Deserialize(t *testing.T) {

	type fields struct {
		WorkingHeight   uint32
		NodeVersion     string
		NodeDownLoadUrl string
		NodeBinHash     common.Uint256
		ForceUpgrade    bool
	}
	type args struct {
		r       io.Reader
		version byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "first_UpgradeCodeInfo_Deserialize",
			fields: fields{
				WorkingHeight:   14,
				NodeVersion:     "123",
				NodeDownLoadUrl: "https://www.google.com/",
				NodeBinHash:     common.Uint256{},
				ForceUpgrade:    false,
			},
			args: args{version: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgradeInfo := &UpgradeCodeInfo{
				WorkingHeight:   tt.fields.WorkingHeight,
				NodeVersion:     tt.fields.NodeVersion,
				NodeDownLoadUrl: tt.fields.NodeDownLoadUrl,
				NodeBinHash:     tt.fields.NodeBinHash,
				ForceUpgrade:    tt.fields.ForceUpgrade,
			}
			buf := new(bytes.Buffer)
			upgradeInfo.Serialize(buf, tt.args.version)
			tt.args.r = buf
			upgradeInfo2 := &UpgradeCodeInfo{}
			if err := upgradeInfo2.Deserialize(tt.args.r, tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("Deserialize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !IsUpgradeCodeInfoEqual(upgradeInfo, upgradeInfo2) {
				t.Errorf("Deserialize() upgradeInfo = %v, upgradeInfo2 %v", upgradeInfo, upgradeInfo2)
			}

		})
	}
}

func IsUpgradeCodeInfoEqual(first, second *UpgradeCodeInfo) bool {
	return first.WorkingHeight == second.WorkingHeight &&
		first.NodeVersion == second.NodeVersion &&
		first.ForceUpgrade == second.ForceUpgrade &&
		first.NodeDownLoadUrl == second.NodeDownLoadUrl &&
		first.NodeBinHash.IsEqual(second.NodeBinHash)
}

func TestUpgradeCodeInfo_Serialize(t *testing.T) {
	type fields struct {
		WorkingHeight   uint32
		NodeVersion     string
		NodeDownLoadUrl string
		NodeBinHash     common.Uint256
		ForceUpgrade    bool
	}
	type args struct {
		version byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantW   string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "first",
			fields: fields{
				WorkingHeight:   14,
				NodeVersion:     "123",
				NodeDownLoadUrl: "https://www.google.com/",
				NodeBinHash:     common.Uint256{},
				ForceUpgrade:    false,
			},
			args:    args{version: 0},
			wantW:   "\u000E\u0000\u0000\u0000\u0003123\u0017https://www.google.com/\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgradeInfo := &UpgradeCodeInfo{
				WorkingHeight:   tt.fields.WorkingHeight,
				NodeVersion:     tt.fields.NodeVersion,
				NodeDownLoadUrl: tt.fields.NodeDownLoadUrl,
				NodeBinHash:     tt.fields.NodeBinHash,
				ForceUpgrade:    tt.fields.ForceUpgrade,
			}
			w := &bytes.Buffer{}
			err := upgradeInfo.Serialize(w, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Serialize() gotW = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
