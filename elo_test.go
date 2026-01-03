package elo

import (
	"math"
	"slices"
	"testing"
	"time"
)

type player struct {
	r float64
	k float64
}

func (p player) Rating() float64 {
	return p.r
}

func (p player) KFactor() float64 {
	return p.k
}

func TestCalculate(t *testing.T) {
	for _, test := range []struct {
		name     string
		r        []float64
		k        []float64
		s        []float64
		expected []float64
	}{
		{
			"no competitors (nil)",
			nil,
			nil,
			nil,
			nil,
		},
		{
			"no competitors (empty)",
			[]float64{},
			[]float64{},
			[]float64{},
			[]float64{},
		},
		{
			"no opponents",
			[]float64{1500},
			[]float64{20},
			[]float64{1},
			[]float64{1500},
		},
		{
			"win",
			[]float64{1500, 1500},
			[]float64{20, 20},
			[]float64{1, 0},
			[]float64{1510, 1490},
		},
		{
			"tie",
			[]float64{1500, 1500},
			[]float64{20, 20},
			[]float64{0.5, 0.5},
			[]float64{1500, 1500},
		},
		{
			"loss",
			[]float64{1500, 1500},
			[]float64{20, 20},
			[]float64{0, 1},
			[]float64{1490, 1510},
		},
		{
			"outmatching win",
			[]float64{1700, 1300},
			[]float64{20, 20},
			[]float64{1, 0},
			[]float64{1701.8, 1298.2},
		},
		{
			"upset tie",
			[]float64{1700, 1300},
			[]float64{20, 20},
			[]float64{0.5, 0.5},
			[]float64{1691.8, 1308.2},
		},
		{
			"upset loss",
			[]float64{1700, 1300},
			[]float64{20, 20},
			[]float64{0, 1},
			[]float64{1681.8, 1318.2},
		},
		{
			"outmatching win vs provisional",
			[]float64{1700, 1300},
			[]float64{20, 40},
			[]float64{1, 0},
			[]float64{1701.8, 1296.4},
		},
		{
			"upset tie vs provisional",
			[]float64{1700, 1300},
			[]float64{20, 40},
			[]float64{0.5, 0.5},
			[]float64{1691.8, 1316.4},
		},
		{
			"upset loss vs provisional",
			[]float64{1700, 1300},
			[]float64{20, 40},
			[]float64{0, 1},
			[]float64{1681.8, 1336.4},
		},
		{
			"3-way normal outcome",
			[]float64{1700, 1500, 1300},
			[]float64{20, 20, 20},
			[]float64{1, 0.5, 0},
			[]float64{1704.8, 1500, 1295.2},
		},
		{
			"3-way tie",
			[]float64{1700, 1500, 1300},
			[]float64{20, 20, 20},
			[]float64{0.5, 0.5, 0.5},
			[]float64{1694.8, 1500, 1305.2},
		},
		{
			"3-way reverse outcome",
			[]float64{1700, 1500, 1300},
			[]float64{20, 20, 20},
			[]float64{0, 0.5, 1},
			[]float64{1684.8, 1500, 1315.2},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			actual, err := Default.Calculate(test.r, test.k, test.s)
			if err != nil {
				t.Errorf("Expected no error got %v", err)
				return
			}
			for i := range test.expected {
				if math.Abs(test.expected[i]-actual[i]) > 0.1 {
					t.Errorf("expected %v got %v", test.expected, actual)
				}
			}
		})
	}

	t.Run("error case", func(t *testing.T) {
		_, err := Default.Calculate([]float64{1500, 1500}, []float64{20, 20}, []float64{1})
		if err == nil {
			t.Errorf("expected error got none")
		}
	})
}

func TestH2H(t *testing.T) {
	t.Run("non-tie", func(t *testing.T) {
		expected, _ := Default.Calculate(
			[]float64{1700, 1300},
			[]float64{20, 20},
			[]float64{1, 0},
		)
		actual := Default.H2H(
			[2]Player{player{1700, 20}, player{1300, 20}},
			[2]float64{35, 20},
		)
		if !slices.Equal(expected, actual[:]) {
			t.Errorf("expected %v got %v", expected, actual)
		}
	})
	t.Run("tie", func(t *testing.T) {
		expected, _ := Default.Calculate(
			[]float64{1700, 1300},
			[]float64{20, 20},
			[]float64{0.5, 0.5},
		)
		actual := Default.H2H(
			[2]Player{player{1700, 20}, player{1300, 20}},
			[2]float64{35, 35},
		)
		if !slices.Equal(expected, actual[:]) {
			t.Errorf("expected %v got %v", expected, actual)
		}
	})
}

func TestRace(t *testing.T) {
	t.Run("non-tie", func(t *testing.T) {
		expected, _ := Default.Calculate(
			[]float64{1700, 1500, 1300},
			[]float64{20, 20, 20},
			[]float64{1, 0.5, 0},
		)
		actual, _ := Default.Race(
			[]Player{player{1700, 20}, player{1500, 20}, player{1300, 20}},
			[]time.Duration{100 * time.Millisecond, 120 * time.Millisecond, 140 * time.Millisecond},
			10*time.Millisecond,
		)
		if !slices.Equal(expected, actual[:]) {
			t.Errorf("expected %v got %v", expected, actual)
		}
	})
	t.Run("tie", func(t *testing.T) {
		expected, _ := Default.Calculate(
			[]float64{1700, 1500, 1300},
			[]float64{20, 20, 20},
			[]float64{0.5, 0.5, 0.50},
		)
		actual, _ := Default.Race(
			[]Player{player{1700, 20}, player{1500, 20}, player{1300, 20}},
			[]time.Duration{100 * time.Millisecond, 100 * time.Millisecond, 100 * time.Millisecond},
			10*time.Millisecond,
		)
		if !slices.Equal(expected, actual[:]) {
			t.Errorf("expected %v got %v", expected, actual)
		}
	})
	t.Run("solo race", func(t *testing.T) {
		expected := []float64{1700}
		actual, _ := Default.Race(
			[]Player{player{1700, 20}},
			[]time.Duration{100 * time.Millisecond},
			10*time.Millisecond,
		)
		if !slices.Equal(expected, actual[:]) {
			t.Errorf("expected %v got %v", expected, actual)
		}
	})
}
