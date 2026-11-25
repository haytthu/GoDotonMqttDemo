package main

import (
	"math"
	"math/cmplx"
	"testing"
)

func TestWaveFFTPrimeLength(t *testing.T) {
	const length = 37
	input := make([]complex128, length)
	for i := range input {
		input[i] = complex(math.Sin(float64(i)), math.Cos(2*float64(i)))
	}
	result := append([]complex128(nil), input...)
	transform := newWaveFFT(length)
	transform(result, false)
	for bin, actual := range result {
		var expected complex128
		for i, value := range input {
			expected += value * cmplx.Rect(1, -2*math.Pi*float64(bin*i)/length)
		}
		if cmplx.Abs(actual-expected) > 1e-9 {
			t.Fatalf("bin %d: got %v, want %v", bin, actual, expected)
		}
	}
	transform(result, true)
	for i, actual := range result {
		if cmplx.Abs(actual-length*input[i]) > 1e-9 {
			t.Fatalf("inverse sample %d: got %v, want %v", i, actual, length*input[i])
		}
	}
}
