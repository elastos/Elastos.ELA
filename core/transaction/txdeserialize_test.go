package transaction

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/common/config"
	"github.com/elastos/Elastos.ELA/core/types/functions"
)

func init() {
	functions.GetTransactionByTxType = GetTransaction
	functions.GetTransactionByBytes = GetTransactionByBytes
	functions.CreateTransaction = CreateTransaction
	functions.GetTransactionParameters = GetTransactionparameters
	config.DefaultParams = *config.GetDefaultParams()
}

func TestDeserializeTransaction(t *testing.T) {
	rawTx := "09630102922EE4D8164E3F8F5638DA4F5CCA8D201CDD0E9D4210DD28EE31E82F4" +
		"57BED202103E6C51AD768F7BD2C46E145BA632717C0126F3FD268089744F2B3733B16C" +
		"7A36D00CA9A3B000000000A000000922EE4D8164E3F8F5638DA4F5CCA8D201CDD0E9D4" +
		"210DD28EE31E82F457BED202103E6C51AD768F7BD2C46E145BA632717C0126F3FD2680" +
		"89744F2B3733B16C7A36D00CA9A3B000000000A0000000100132D39383533343232303" +
		"2363432393636353433016694FD731DB00A8727F8B9D710530DDDF16D9448A583486D9" +
		"B7049BFD2EBF99200000000000001B037DB964A231458D2D6FFD5EA18944C4F90E63D5" +
		"47C5D3B9874DF66A4EAD0A300E1F5050000000000000000214FFBC4FB3B3C30A626A3B" +
		"298BFA392A0121D42490000000000014140700AF98C74BC9EB4AD63A2AFF6077904062" +
		"D57E0F37B18A3DD40E0741D566B1D75E5A373A0EB357EE8381F4C11DD34E9E02058F2D" +
		"5BE9B89AEF942C0DE3204152321037F3CAEDE72447B6082C1E8F7705FFD1ED6E24F348" +
		"130D34CBC7C0A35C9E993F5AC"
	data, err := common.HexStringToBytes(rawTx)
	if err != nil {
		fmt.Println("err:", err)

	}
	r := bytes.NewReader(data)
	tx, err := functions.GetTransactionByBytes(r)
	if err != nil {
		fmt.Println("err:", err)
	}
	err = tx.Deserialize(r)
	if err != nil {
		fmt.Println("err:", err)
	}
}
