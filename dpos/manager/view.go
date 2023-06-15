// Copyright (c) 2017-2020 The Elastos Foundation
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
//

package manager

import (
	"bytes"
	"math"
	"time"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/dpos/log"
	"github.com/elastos/Elastos.ELA/dpos/state"
)

type ViewListener interface {
	OnViewChanged(isOnDuty bool)
}

type view struct {
	publicKey     []byte
	signTolerance time.Duration
	viewStartTime time.Time
	isDposOnDuty  bool
	arbitrators   state.Arbitrators

	listener ViewListener
}

func (v *view) IsOnDuty() bool {
	return v.isDposOnDuty
}

func (v *view) SetOnDuty(onDuty bool) {
	v.isDposOnDuty = onDuty
}

func (v *view) GetViewStartTime() time.Time {
	return v.viewStartTime
}

func (v *view) ResetView(t time.Time) {
	v.viewStartTime = t
}

func (v *view) ChangeView(viewOffset *uint32, now time.Time) {
	//offset, offsetTime := v.calculateOffsetTimeV0(v.viewStartTime, now)
	//*viewOffset += uint32(offset)

	arbitersCount := v.arbitrators.GetArbitersCount()

	offset, offsetTime := v.calculateOffsetTimeV1(*viewOffset, v.viewStartTime, now, uint32(arbitersCount))
	*viewOffset = uint32(offset)

	//offset, offsetTime := v.calculateOffsetTimeV2(*viewOffset, v.viewStartTime, now, uint32(arbitersCount))
	//*viewOffset = uint32(offset)

	v.viewStartTime = now.Add(-offsetTime)

	if offset > 0 {
		currentArbiter := v.arbitrators.GetNextOnDutyArbitrator(*viewOffset)

		v.isDposOnDuty = bytes.Equal(currentArbiter, v.publicKey)
		log.Info("current onduty arbiter:",
			common.BytesToHexString(currentArbiter))

		v.listener.OnViewChanged(v.isDposOnDuty)
	}
}

func (v *view) calculateOffsetTimeV0(startTime time.Time,
	now time.Time) (uint32, time.Duration) {
	duration := now.Sub(startTime)
	offset := duration / v.signTolerance
	offsetTime := duration % v.signTolerance

	return uint32(offset), offsetTime
}

func (v *view) calculateOffsetTimeV1(currentViewOffset uint32, startTime time.Time,
	now time.Time, arbitersCount uint32) (uint32, time.Duration) {
	step := float64(3)
	duration := now.Sub(startTime)
	currentOffset := currentViewOffset

	offsetSeconds := time.Duration(5*math.Pow(step, float64(currentOffset/arbitersCount))) * time.Second
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		offsetSeconds = time.Duration(5*math.Pow(step, float64(currentOffset/arbitersCount))) * time.Second
	}

	return currentOffset, duration
}

func (v *view) calculateOffsetTimeV2(currentViewOffset uint32, startTime time.Time,
	now time.Time, arbitersCount uint32) (uint32, time.Duration) {
	duration := now.Sub(startTime)
	currentOffset := currentViewOffset

	offsetSeconds := time.Duration(5+(currentOffset-arbitersCount)) * time.Second
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		offsetSeconds = time.Duration(5+(currentOffset-arbitersCount)) * time.Second
	}

	return currentOffset, duration
}

func (v *view) TryChangeView(viewOffset *uint32, now time.Time) bool {
	if now.After(v.viewStartTime.Add(v.signTolerance)) {
		log.Info("[TryChangeView] succeed")
		v.ChangeView(viewOffset, now)
		return true
	}
	return false
}

func (v *view) GetViewInterval() time.Duration {
	return v.signTolerance
}
