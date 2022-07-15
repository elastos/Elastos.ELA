package common

import (
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	t1 := TransactionHistoryDisplay{
		Height: 10,
	}
	t2 := TransactionHistoryDisplay{
		Height: 2,
	}
	t0 := TransactionHistorySorter{
		t1,
		t2,
	}
	sort.Sort(t0)
}
