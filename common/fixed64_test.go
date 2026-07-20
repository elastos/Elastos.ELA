// Copyright (c) 2026 The Elastos Foundation
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package common

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestFixed64CheckedArithmeticBoundaries(t *testing.T) {
	testCases := []struct {
		name      string
		operation func(Fixed64, Fixed64) (Fixed64, error)
		left      Fixed64
		right     Fixed64
		expected  Fixed64
		hasError  bool
	}{
		{"add max boundary", AddFixed64, math.MaxInt64 - 1, 1, math.MaxInt64, false},
		{"add positive overflow", AddFixed64, math.MaxInt64, 1, 0, true},
		{"add min boundary", AddFixed64, math.MinInt64 + 1, -1, math.MinInt64, false},
		{"add negative overflow", AddFixed64, math.MinInt64, -1, 0, true},
		{"subtract min boundary", SubtractFixed64, math.MinInt64 + 1, 1, math.MinInt64, false},
		{"subtract negative overflow", SubtractFixed64, math.MinInt64, 1, 0, true},
		{"subtract max boundary", SubtractFixed64, math.MaxInt64 - 1, -1, math.MaxInt64, false},
		{"subtract positive overflow", SubtractFixed64, math.MaxInt64, -1, 0, true},
		{"multiply max boundary", MultiplyFixed64, math.MaxInt64, 1, math.MaxInt64, false},
		{"multiply positive overflow", MultiplyFixed64, math.MaxInt64, 2, 0, true},
		{"multiply min boundary", MultiplyFixed64, math.MinInt64, 1, math.MinInt64, false},
		{"multiply min by negative one", MultiplyFixed64, math.MinInt64, -1, 0, true},
		{"multiply negative overflow", MultiplyFixed64, math.MinInt64, 2, 0, true},
		{"multiply by zero", MultiplyFixed64, math.MinInt64, 0, 0, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := testCase.operation(testCase.left, testCase.right)
			if testCase.hasError {
				if !errors.Is(err, ErrFixed64Overflow) {
					t.Fatalf("expected Fixed64 overflow, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != testCase.expected {
				t.Fatalf("expected %d, got %d", testCase.expected, actual)
			}
		})
	}
}

func FuzzAddFixed64AgainstBigInt(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(math.MaxInt64), int64(1))
	f.Add(int64(math.MinInt64), int64(-1))

	f.Fuzz(func(t *testing.T, left, right int64) {
		expected := new(big.Int).Add(big.NewInt(left), big.NewInt(right))
		actual, err := AddFixed64(Fixed64(left), Fixed64(right))
		if !expected.IsInt64() {
			if !errors.Is(err, ErrFixed64Overflow) {
				t.Fatalf("expected overflow for %d + %d, got %v", left, right, err)
			}
			return
		}
		if err != nil || int64(actual) != expected.Int64() {
			t.Fatalf("unexpected result for %d + %d: %d, %v", left, right,
				actual, err)
		}
	})
}

func FuzzSubtractFixed64AgainstBigInt(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(math.MinInt64), int64(1))
	f.Add(int64(math.MaxInt64), int64(-1))

	f.Fuzz(func(t *testing.T, left, right int64) {
		expected := new(big.Int).Sub(big.NewInt(left), big.NewInt(right))
		actual, err := SubtractFixed64(Fixed64(left), Fixed64(right))
		if !expected.IsInt64() {
			if !errors.Is(err, ErrFixed64Overflow) {
				t.Fatalf("expected overflow for %d - %d, got %v", left, right, err)
			}
			return
		}
		if err != nil || int64(actual) != expected.Int64() {
			t.Fatalf("unexpected result for %d - %d: %d, %v", left, right,
				actual, err)
		}
	})
}

func FuzzMultiplyFixed64AgainstBigInt(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(math.MaxInt64), int64(2))
	f.Add(int64(math.MinInt64), int64(-1))

	f.Fuzz(func(t *testing.T, left, right int64) {
		expected := new(big.Int).Mul(big.NewInt(left), big.NewInt(right))
		actual, err := MultiplyFixed64(Fixed64(left), Fixed64(right))
		if !expected.IsInt64() {
			if !errors.Is(err, ErrFixed64Overflow) {
				t.Fatalf("expected overflow for %d * %d, got %v", left, right, err)
			}
			return
		}
		if err != nil || int64(actual) != expected.Int64() {
			t.Fatalf("unexpected result for %d * %d: %d, %v", left, right,
				actual, err)
		}
	})
}
