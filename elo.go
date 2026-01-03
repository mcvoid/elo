package elo

import (
	"errors"
	"math"
	"slices"
	"time"
)

var Default = Elo{400}

// A representation of a player with a rating.
type Player interface {
	// An indicator of the player's strength
	Rating() float64
	// A "swing factor" - how much the rating will change relative to the expected outcome.
	// For example, for a KFactor factor of 32, if someone with a predicted outcome of 0
	// instead wins 1-0, their rating will increase by 32 (the maximum for that KFactor value).
	KFactor() float64
}

// An instance of an Elo rating system.
type Elo struct {
	// The rating difference between 2 players which would one player 10-to-1 odds of winning.
	Base float64
}

// Calculate new Elo ratings. Returns an error if the given slices aren't the same size.
func (e Elo) Calculate(ratings, kFactors, normalizedScores []float64) (newRatings []float64, err error) {
	// we're going to be using this a lot (11 times), so have some shorthand for readability
	l := len(ratings)

	// misshapen slices
	if l != len(kFactors) || l != len(normalizedScores) {
		return nil, errors.New("ratings, kValues, and normalizedScores should be the same length")
	}

	// no competitors == no results
	if l == 0 {
		return slices.Clone(ratings), nil
	}
	// no opponent == no rating change
	if l == 1 {
		return slices.Clone(ratings), nil
	}

	// first we need sorted versions in order of score
	// keeping a list of indices to reverse the process
	idx := make([]int, l)
	for i := range l {
		idx[i] = i
	}
	slices.SortStableFunc(idx, cmp(normalizedScores))

	// use the index lookup to create new slices in score order
	r, k, s := make([]float64, l), make([]float64, l), make([]float64, l)
	for i, j := range idx {
		r[i] = ratings[j]
		k[i] = kFactors[j]
		s[i] = normalizedScores[j]
	}

	// do the actual Elo calulations
	diffs := make([]float64, l)
	for a := 0; a < l-1; a++ {
		b := a + 1
		Qa, Qb := math.Pow(10, r[a]/e.Base), math.Pow(10, r[b]/e.Base)
		Ea, Eb := Qa/(Qa+Qb), Qb/(Qa+Qb)
		diffs[a] += k[a] * (s[a] - Ea)
		diffs[b] += k[b] * (s[b] - Eb)
	}

	// copy the ratings into results
	newRatings = slices.Clone(ratings)
	// adjust the results by the calculated diff
	for i, j := range idx {
		// have to reverse the sort using the stored indices
		// (this is the opposite of r[i] = ratings[j])
		newRatings[j] += diffs[i]
	}

	return newRatings, nil
}

// Calculate the new Elo ratings for a head-to-head (1-on-1) match. Will normalize the score for you if it's not already normalized.
func (e Elo) H2H(players [2]Player, scores [2]float64) [2]float64 {
	newRatings, _ := e.FFA(players[:], scores[:])
	results := [2]float64{}
	copy(results[:], newRatings)
	return results
}

// Calculate the new Elo ratings for a free-for-all (many vs many). Will normalize scores between [lowest, highest].
// Returns an error if the length of s doesn't match the number of players.
func (e Elo) FFA(players []Player, scores []float64) ([]float64, error) {
	r := make([]float64, len(players))
	k := make([]float64, len(players))
	for i := range players {
		r[i] = players[i].Rating()
		k[i] = players[i].KFactor()
	}

	// don't mess with input
	s := slices.Clone(scores)
	normalize(s)
	return e.Calculate(r, k, s)
}

// Calculate the new Elo ratings for a race. Will turn the times into normalized scores accurate to the step duration.
// Returns an error if the length of times doesn't match the number of players.
func (e Elo) Race(players []Player, times []time.Duration, step time.Duration) ([]float64, error) {
	s := make([]float64, len(times))
	for i := range times {
		s[i] = float64(times[i] / step)
	}
	return e.Golf(players, s)
}

// Calculate the new Elo ratings for golf (free-for-all, lowest score wins). Will normalize scores between [lowest, highest].
// Returns an error if the length of s doesn't match the number of players.
func (e Elo) Golf(players []Player, scores []float64) ([]float64, error) {
	r := make([]float64, len(players))
	k := make([]float64, len(players))
	for i := range players {
		r[i] = players[i].Rating()
		k[i] = players[i].KFactor()
	}

	// don't mess with input
	s := slices.Clone(scores)
	normalize(s)
	// lower scores are better: reverse the normals
	for i, n := range s {
		s[i] = 1 - n
	}
	return e.Calculate(r, k, s)
}

func normalize(s []float64) {
	// don't bother normalizing less than 2 - it won't affect the calculation
	if len(s) < 2 {
		return
	}

	// normalize ties (no min or max) to 0.5
	isEqual := true
	for i := 0; i < len(s)-1; i++ {
		if math.Abs(s[i]-s[i+1]) > 1e-9 {
			isEqual = false
			break
		}
	}
	if isEqual {
		for i := range s {
			s[i] = 0.5
		}
		return
	}

	// Truncating to the min and scaling by 1/(max-min) does some nice things
	// Last place is 0, first place is 1, so it works for head-to-head and bigger groups
	// do it in 2 passes to make sure float calculations are accurate-ish (max / max == 1.0 exactly)
	min := slices.Min(s)
	for i, f := range s {
		s[i] = f - min
	}
	max := slices.Max(s) // this does 1 more pass than needed, but is more readable
	for i, f := range s {
		s[i] = f / max
	}
}

func cmp(s []float64) func(a, b int) int {
	return func(a, b int) int {
		sA := s[a]
		sB := s[b]
		if sA < sB {
			return -1
		}
		if sA > sB {
			return 1
		}
		return 0
	}
}
