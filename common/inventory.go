package common

type InventoryType byte

const (
	TRANSACTION	InventoryType = 0x01
	BLOCK		InventoryType = 0x02
	CONSENSUS	InventoryType = 0xe0
)
