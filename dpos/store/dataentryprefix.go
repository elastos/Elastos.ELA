package store

// DataEntryPrefix
type DataEntryPrefix byte

const (
	// DPOS
	DPOSDutyChangedCount  DataEntryPrefix = 0x11
	DPOSCurrentArbiters   DataEntryPrefix = 0x12
	DPOSCurrentCandidates DataEntryPrefix = 0x13
	DPOSNextArbiters      DataEntryPrefix = 0x14
	DPOSNextCandidates    DataEntryPrefix = 0x15
	DPOSDirectPeers       DataEntryPrefix = 0x16
	DPOSEmergencyData     DataEntryPrefix = 0x17
)
