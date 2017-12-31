package pow

import (
    "ELA/core/ledger"
)

const (
    MSGHASKBLOCK = "hashblock"
    MSGHASKTX    = "hashtx"
)

func (pow *PowService) ZMQClientSend(MsgBlock ledger.Block) {

}

func (pow *PowService) ZMQServer() {

}
