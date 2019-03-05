// This file defines the legacy logic in past versions.
package state

// getOnDutyArbiterV0 defines the legacy version 0 of getting arbiter.
func (d *DutyState) getOnDutyArbiterV0(offset uint32) []byte {
	index := (d.bestHeight() + offset) % uint32(len(d.orgArbiters))
	return d.orgArbiters[index]
}

// getArbitersV0 defines the legacy version 0 of getting arbiters.
func (d *DutyState) getArbitersV0() [][]byte {
	return d.orgArbiters
}
