package manager

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/elastos/Elastos.ELA/utils/test"
)

func TestView_ChangeViewV1(t *testing.T) {
	test.SkipShort(t)
	print("[")
	for step := uint32(1); step <= 50; step++ {
		for i := 0; i < 100; i++ {
			// Create an array of 36 uint32 values
			arr := [36]uint32{}

			// Fill the array with random values from 0 to 36
			for j := 0; j < len(arr); j++ {
				arr[j] = uint32(rand.Intn(10))
			}
			ti, _ := changeViewToSameV1(arr, step)
			fmt.Printf("[%.2f,%d],", float64(step)/10, int(ti.Seconds()))
		}
	}
	print("]")
}

func TestView_ChangeViewV2(t *testing.T) {
	test.SkipShort(t)
	print("[")
	for step := uint32(2); step <= 36; step++ {
		for i := 0; i < 100; i++ {
			// Create an array of 36 uint32 values
			arr := make([]offsetIndex, 36, 36)
			// Fill the array with random values from 0 to 36
			for i := 0; i < 36; i++ {
				arr[i] = offsetIndex{
					Offset: uint32(rand.Intn(36)),
					Index:  i,
				}
			}
			ti, _ := changeViewToSameV2(arr, step)
			print("[", step, ",", int(ti.Seconds()), "],")
		}
	}
	print("]")
}

func TestView_ChangeViewV3(t *testing.T) {
	test.SkipShort(t)
	addTime := uint32(1)
	offsetTime := 60 * time.Second
	arbitersCount := 36
	print("[")
	for step := uint32(2); step <= 36; step++ {
		currentOffset, _, _ := calculateOffsetTimeV3(addTime, step, 0, offsetTime, uint32(arbitersCount))

		for i := 0; i < 100; i++ {
			// Create an array of 36 uint32 values
			arr := make([]offsetIndex, arbitersCount, arbitersCount)
			// Fill the array with random values from 0 to arbitersCount
			for i := 0; i < arbitersCount; i++ {
				arr[i] = offsetIndex{
					Offset: uint32(rand.Intn(int(currentOffset) + 1)),
					Index:  i,
				}
			}
			ti, _ := changeViewToSameV3(arr, addTime, step)
			print("[", step, ",", int(ti.Seconds()), "],")
		}
	}
	print("]")
}

func TestView_ChangeViewV3_Special(t *testing.T) {
	test.SkipShort(t)
	addTime := uint32(1)
	offsetTime := 60 * time.Second
	arbitersCount := 36
	print("[")
	for step := uint32(2); step <= 36; step++ {
		currentOffset, _, _ := calculateOffsetTimeV3(addTime, step, 0, offsetTime, uint32(arbitersCount))

		// Create an array of 36 uint32 values
		arr := make([]offsetIndex, arbitersCount, arbitersCount)
		viewOffset1 := uint32(0)
		viewOffset2 := currentOffset
		for i := 0; i < arbitersCount; i++ {
			if i < arbitersCount/2 {
				arr[i] = offsetIndex{
					Offset: viewOffset1,
					Index:  i,
				}
			} else {
				arr[i] = offsetIndex{
					Offset: viewOffset2,
					Index:  i,
				}
			}

		}

		ti, _ := changeViewToSameV3(arr, addTime, step)
		print("[", step, ",", int(ti.Seconds()), "],")
	}
	print("]")
}

func TestView_ChangeViewV4(t *testing.T) {
	test.SkipShort(t)
	addTime := uint32(1)
	offsetTime := 3600 * time.Second
	arbitersCount := 36
	print("[")
	for step := uint32(2); step <= 36; step++ {
		currentOffset, _, _ := calculateOffsetTimeV4(addTime, step, 0, offsetTime, uint32(arbitersCount))

		for i := 0; i < 100; i++ {
			// Create an array of 36 uint32 values
			arr := make([]offsetIndex, arbitersCount, arbitersCount)
			// Fill the array with random values from 0 to arbitersCount
			for i := 0; i < arbitersCount; i++ {
				arr[i] = offsetIndex{
					Offset: uint32(rand.Intn(int(currentOffset) + 1)),
					Index:  i,
				}
			}
			ti, _ := changeViewToSameV4(arr, addTime, step)
			print("[", step, ",", int(ti.Seconds()), "],")
		}
	}
	print("]")
}

func TestView_ChangeViewV4_Special(t *testing.T) {
	test.SkipShort(t)
	addTime := uint32(3)
	offsetTime := 7200 * time.Second
	arbitersCount := 36
	print("[")
	for step := uint32(2); step <= 36; step++ {
		currentOffset, _, _ := calculateOffsetTimeV4(addTime, step, 0, offsetTime, uint32(arbitersCount))

		// Create an array of 36 uint32 values
		arr := make([]offsetIndex, arbitersCount, arbitersCount)
		viewOffset1 := uint32(0)
		viewOffset2 := currentOffset
		for i := 0; i < arbitersCount; i++ {
			if i < arbitersCount/2 {
				arr[i] = offsetIndex{
					Offset: viewOffset1,
					Index:  i,
				}
			} else {
				arr[i] = offsetIndex{
					Offset: viewOffset2,
					Index:  i,
				}
			}

		}

		ti, _ := changeViewToSameV4(arr, addTime, step)
		print("[", step, ",", int(ti.Seconds()), "],")
	}
	print("]")
}

func calculateOffsetTimeV2(step float64, currentViewOffset uint32, duration time.Duration, arbitersCount uint32) (uint32, time.Duration, time.Duration) {
	currentOffset := currentViewOffset

	offsetSeconds := time.Duration(5*math.Pow(step, float64(currentOffset/arbitersCount))) * time.Second
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		offsetSeconds = time.Duration(5*math.Pow(step, float64(currentOffset/arbitersCount))) * time.Second
	}

	return currentOffset, duration, offsetSeconds
}

func calculateOffsetTimeV3(addTime uint32, step uint32, currentViewOffset uint32, duration time.Duration, arbitersCount uint32) (uint32, time.Duration, time.Duration) {
	currentOffset := currentViewOffset

	var offsetSeconds time.Duration
	if currentOffset < arbitersCount {
		offsetSeconds = 5 * time.Second
	} else {
		offsetSeconds = time.Duration(5+(currentOffset-arbitersCount)*addTime*
			uint32(math.Pow(float64(step), float64(currentOffset/arbitersCount)))) * time.Second
	}
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		if currentOffset < arbitersCount {
			offsetSeconds = 5 * time.Second
		} else {
			offsetSeconds = time.Duration(5+(currentOffset-arbitersCount)*addTime*
				uint32(math.Pow(float64(step), float64(currentOffset/arbitersCount)))) * time.Second
		}
	}

	return currentOffset, duration, offsetSeconds
}

func calculateOffsetTimeV4(addTime uint32, step uint32, currentViewOffset uint32, duration time.Duration, arbitersCount uint32) (uint32, time.Duration, time.Duration) {
	currentOffset := currentViewOffset

	var offsetSeconds time.Duration
	if currentOffset < arbitersCount {
		offsetSeconds = 5 * time.Second
	} else {
		offsetSeconds = time.Duration(5*uint32(math.Pow(float64(step),
			float64(currentOffset/arbitersCount)))+(currentOffset-arbitersCount)*addTime) * time.Second
	}
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		if currentOffset < arbitersCount {
			offsetSeconds = 5 * time.Second
		} else {
			offsetSeconds = time.Duration(5*uint32(math.Pow(float64(step),
				float64(currentOffset/arbitersCount)))+(currentOffset-arbitersCount)*addTime) * time.Second
		}
	}

	return currentOffset, duration, offsetSeconds
}

type viewTime struct {
	Times int
	K     map[int]struct{}
}

type offsetIndex struct {
	Offset uint32
	Index  int
}

func changeViewToSameV2(viewOffsets []offsetIndex, step uint32) (sameTime time.Duration, consensusTime time.Duration) {

	sort.Slice(viewOffsets, func(i, j int) bool {
		return viewOffsets[i].Offset > viewOffsets[j].Offset
	})

	// 2/3+1 same view offset
	smallestK := math.MaxInt32
	for currentTime := 5 * time.Second; ; currentTime += 5 * time.Second {
		offsets := make(map[uint32]viewTime)
		var offsetSeconds time.Duration

		for k, v := range viewOffsets {
			var offset uint32
			offset, _, offsetSeconds = calculateOffsetTimeV2(float64(step), v.Offset, currentTime, uint32(len(viewOffsets)))
			if offsets[offset].Times >= 24 {
				sameTime = currentTime
				if smallestK > v.Index {
					smallestK = v.Index
				}

				if smallestK != math.MaxInt32 {
					randomIndex := rand.Intn(36)
					consensusTime = sameTime + time.Duration(randomIndex)*offsetSeconds
					return
				}
			}

			if ot, ok := offsets[offset]; ok {
				ot.K[k] = struct{}{}
				offsets[offset] = viewTime{
					Times: ot.Times + 1,
					K:     ot.K,
				}
			} else {
				offsets[offset] = viewTime{
					Times: 1,
					K:     make(map[int]struct{}),
				}
				offsets[offset].K[k] = struct{}{}
			}
		}
	}
}

func calculateOffsetTimeV1(step uint32, currentViewOffset uint32, duration time.Duration, arbitersCount uint32) (uint32, time.Duration, time.Duration) {
	currentOffset := currentViewOffset

	var offsetSeconds time.Duration
	if currentOffset < arbitersCount {
		offsetSeconds = 5 * time.Second
	} else {
		offsetSeconds = time.Duration(5000+float64(currentOffset-arbitersCount)*(float64(step)*1000/10)) * time.Millisecond
	}
	for duration >= offsetSeconds {
		currentOffset++
		duration -= offsetSeconds
		if currentOffset < arbitersCount {
			offsetSeconds = 5 * time.Second
		} else {
			offsetSeconds = time.Duration(5+(currentOffset-arbitersCount)*step) * time.Second
		}
	}

	return currentOffset, duration, offsetSeconds
}

func changeViewToSameV1(viewOffsets [36]uint32, step uint32) (sameTime time.Duration, consensusTime time.Duration) {

	// 2/3+1 same view offset
	for currentTime := 5 * time.Second; ; currentTime += 5 * time.Second {
		offsets := make(map[uint32]viewTime)
		var offsetSeconds time.Duration
		for k, v := range viewOffsets {
			var offset uint32
			offset, _, offsetSeconds = calculateOffsetTimeV1(step, v, currentTime, 36)
			if offsets[offset].Times >= 24 {
				sameTime = currentTime

				smallestK := math.MaxInt32
				for i := 1; i <= len(viewOffsets); i++ {
					if _, ok := offsets[offset].K[i]; !ok {
						if smallestK > i {
							smallestK = i
							break
						}
					}
				}
				randomIndex := rand.Intn(36)
				consensusTime = sameTime + time.Duration(randomIndex)*offsetSeconds

				return
			}

			if ot, ok := offsets[offset]; ok {
				ot.K[k] = struct{}{}
				offsets[offset] = viewTime{
					Times: ot.Times + 1,
					K:     ot.K,
				}
			} else {
				offsets[offset] = viewTime{
					Times: 1,
					K:     make(map[int]struct{}),
				}
				offsets[offset].K[k] = struct{}{}
			}
		}

	}
}

func changeViewToSameV3(viewOffsets []offsetIndex, addTime uint32, step uint32) (sameTime time.Duration, consensusTime time.Duration) {

	sort.Slice(viewOffsets, func(i, j int) bool {
		return viewOffsets[i].Offset > viewOffsets[j].Offset
	})

	// 2/3+1 same view offset
	smallestK := math.MaxInt32
	for currentTime := 5 * time.Second; ; currentTime += 5 * time.Second {
		offsets := make(map[uint32]viewTime)
		var offsetSeconds time.Duration

		for k, v := range viewOffsets {
			var offset uint32
			offset, _, offsetSeconds = calculateOffsetTimeV3(addTime, step, v.Offset, currentTime, uint32(len(viewOffsets)))
			if offsets[offset].Times >= 24 {
				sameTime = currentTime
				if smallestK > v.Index {
					smallestK = v.Index
				}

				if smallestK != math.MaxInt32 {
					randomIndex := rand.Intn(36)
					consensusTime = sameTime + time.Duration(randomIndex)*offsetSeconds
					return
				}
			}

			if ot, ok := offsets[offset]; ok {
				ot.K[k] = struct{}{}
				offsets[offset] = viewTime{
					Times: ot.Times + 1,
					K:     ot.K,
				}
			} else {
				offsets[offset] = viewTime{
					Times: 1,
					K:     make(map[int]struct{}),
				}
				offsets[offset].K[k] = struct{}{}
			}
		}
	}
}

func changeViewToSameV4(viewOffsets []offsetIndex, addTime uint32, step uint32) (sameTime time.Duration, consensusTime time.Duration) {

	sort.Slice(viewOffsets, func(i, j int) bool {
		return viewOffsets[i].Offset > viewOffsets[j].Offset
	})

	// 2/3+1 same view offset
	smallestK := math.MaxInt32
	for currentTime := 5 * time.Second; ; currentTime += 5 * time.Second {
		offsets := make(map[uint32]viewTime)
		var offsetSeconds time.Duration

		for k, v := range viewOffsets {
			var offset uint32
			offset, _, offsetSeconds = calculateOffsetTimeV4(addTime, step, v.Offset, currentTime, uint32(len(viewOffsets)))
			if offsets[offset].Times >= 24 {
				sameTime = currentTime
				if smallestK > v.Index {
					smallestK = v.Index
				}

				if smallestK != math.MaxInt32 {
					randomIndex := rand.Intn(36)
					consensusTime = sameTime + time.Duration(randomIndex)*offsetSeconds
					return
				}
			}

			if ot, ok := offsets[offset]; ok {
				ot.K[k] = struct{}{}
				offsets[offset] = viewTime{
					Times: ot.Times + 1,
					K:     ot.K,
				}
			} else {
				offsets[offset] = viewTime{
					Times: 1,
					K:     make(map[int]struct{}),
				}
				offsets[offset].K[k] = struct{}{}
			}
		}
	}
}
