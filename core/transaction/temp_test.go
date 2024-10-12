package transaction

import (
	"fmt"
	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/utils"
	"testing"
)

func Test_ReverTxID(t *testing.T) {
	str := "65dd4737e9a85030d341653ea21005bb132200a97ad8cc8555f4c28ae2e16d71"
	txHash, _ := common.Uint256FromHexString(str)
	fmt.Println(common.ToReversedString(*txHash))

	code, _ := common.HexStringToBytes("2103997349de5629299fd2b8d255c99d6b2047c6fcfa0237e9d1b07e5ac8db45f310ac")
	addr, _ := utils.GetAddressByCode(code)
	fmt.Println("addr:", addr)

	saddr, _ := utils.GetStakeAddressByCode(code)
	fmt.Println("saddr:", saddr)
}
