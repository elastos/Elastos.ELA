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

const (
	ChangeViewAddStep = uint32(3)
	ChangeViewMulStep = uint32(20)
)

type ViewListener interface {
	OnViewChanged(isOnDuty bool)
}

type view struct {
	publicKey          []byte
	signTolerance      time.Duration
	viewStartTime      time.Time
	isDposOnDuty       bool
	changeViewV1Height uint32
	arbitrators        state.Arbitrators

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
	offset, offsetTime := v.calculateOffsetTimeV0(v.viewStartTime, now)
	*viewOffset += offset

	v.viewStartTime = now.Add(-offsetTime)

	if offset > 0 {
		currentArbiter := v.arbitrators.GetNextOnDutyArbitrator(*viewOffset)

		v.isDposOnDuty = bytes.Equal(currentArbiter, v.publicKey)
		log.Info("current onduty arbiter:",
			common.BytesToHexString(currentArbiter))

		v.listener.OnViewChanged(v.isDposOnDuty)
	}
}

func (v *view) ChangeViewV1(viewOffset *uint32, now time.Time) bool {
	arbitersCount := v.arbitrators.GetArbitersCount()

	offset, offsetTime := v.calculateOffsetTimeV1(*viewOffset, v.viewStartTime, now, uint32(arbitersCount))
	if offset == *viewOffset {
		return false
	}
	log.Info("ChangeView succeed, offset from:", *viewOffset, "to:", offset)

	*viewOffset = offset
	v.viewStartTime = now.Add(-offsetTime)

	if offset > 0 {
		currentArbiter := v.arbitrators.GetNextOnDutyArbitrator(*viewOffset)

		v.isDposOnDuty = bytes.Equal(currentArbiter, v.publicKey)
		log.Info("current onduty arbiter:",
			common.BytesToHexString(currentArbiter))

		v.listener.OnViewChanged(v.isDposOnDuty)
	}

	return true
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

	currentOffset := currentViewOffset
	duration := now.Sub(startTime)

	var offsetSeconds time.Duration
	if currentOffset < arbitersCount {
		offsetSeconds = 5 * time.Second
	} else {
		// view 18:  5+ 0*3 * 20^1 = 5
		// view 19:  5+ 1*3 * 20^1 = 65
		// view 20:  5+ 2*3 * 20^1 = 125
		// view 21:  5+ 3*3 * 20^1 = 185
		// view 22:  5+ 4*3 * 20^1 = 245
		// view 23:  5+ 5*3 * 20^1 = 305
		// view 24:  5+ 6*3 * 20^1 = 365
		// view 25:  5+ 7*3 * 20^1 = 425
		// view 26:  5+ 8*3 * 20^1 = 485
		// view 27:  5+ 9*3 * 20^1 = 545
		// view 28:  5+ 10*3 * 20^1 = 605
		// view 29:  5+ 11*3 * 20^1 = 665
		// view 30:  5+ 12*3 * 20^1 = 725
		// view 31:  5+ 13*3 * 20^1 = 785
		// view 32:  5+ 14*3 * 20^1 = 845
		// view 33:  5+ 15*3 * 20^1 = 905
		// view 34:  5+ 16*3 * 20^1 = 965
		// view 35:  5+ 17*3 * 20^1 = 1025
		offsetSeconds = time.Duration(5+(currentOffset-arbitersCount)*ChangeViewAddStep*
			uint32(math.Pow(float64(ChangeViewMulStep), float64(currentOffset/arbitersCount)))) * time.Second
	}
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		if currentOffset < arbitersCount {
			offsetSeconds = 5 * time.Second
		} else {
			offsetSeconds = time.Duration(5+(currentOffset-arbitersCount)*ChangeViewAddStep*
				uint32(math.Pow(float64(ChangeViewMulStep), float64(currentOffset/arbitersCount)))) * time.Second
		}
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

func (v *view) TryChangeViewV1(viewOffset *uint32, now time.Time) bool {
	if now.After(v.viewStartTime.Add(v.signTolerance)) {
		return v.ChangeViewV1(viewOffset, now)
	}
	return false
}

func (v *view) GetViewInterval() time.Duration {
	return v.signTolerance
}
