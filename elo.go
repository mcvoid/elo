// Calculates ratings using the Elo system, in several different formats.
//
// ** Installation
//
// `go get github.com/mcvoid/elo`
//
// # Usage
//
//		// A head-to-head match between two players
//		newRatings := elo.Default.H2H(
//			[...]Player{p1, p2}, // the players in the match
//	 	    [...]float64{25, 7}, // their scores
//		)
//
// # How it works
//
// The mathematical basis for adjusting ratings is here: https://en.wikipedia.org/wiki/Elo_rating_system#Mathematical_details.
// Generally, when comparing two people, the ratings are compared on a Bell
// curve of a given width (default: 400) to calculate an expected score, normalized
// to the range [0,1]. It is then compared to the normalized actual score, and scaled
// by some factor K, unique to each player, to increase or decrease the rating.
//
// Important user-supplied constants:
//   - Base: By default, a player with a rating of 400 points higher than the opponent is expected to win ten times as often. This leads to a variation of a couple thousand from the starting rating, depending on the skill spread. A league that wants a smaller or larger point spread can adjust this factor.
//   - KFactor: A high K value represents uncertainty in the rating of a player. Each player
//
// has their own K factor. Leagues typically give a high K to newer players and lower it
// as they play more games. A high L leads to very swingy ratings.
//
// When there's more than two players, it's modeled as if everyone is sorted by their score, and they
// are treated as if they played a match against the person who finished just ahead and just behind them.
//
// # Special Cases
//
// The following situations are provided to make it easier for common scenarios:
//
//   - H2H: Head-to-head matches. One player vs another, or one team vs another.
//   - FFA: Free-for-all. Two or more players, high score wins. 1st place gets 1, last place gets 0, the rest get somewhere in the middle relative to where their score sat between the high and low scores.
//   - Golf: Like free-for-all, but lowest score wins.
//   - Race: Like Golf, but times are input and converted to scores automatically.
//   - Place: Provided with a list of who finished first, second, etc, points are given based on place.
//
// # General Case
//
// If none of the above scenarios are appropriate, the general calculator is available. Provide it with
// a list of ratings along with a corresponding list of k-factors and scores and it will give you the
// corresponding list of new ratings.\
//
//	expected, err := elo.Default.Calculate(
//		[]float64{1700, 1500, 1300},
//		[]float64{20, 20, 20},
//		[]float64{1, 0.5, 0},
//	)
//
// # Change of Base
//
// Manipulate the rating spread by using a different base like so:
//
//	expected, err := elo.Elo{1000}.Calculate(
//		[]float64{1700, 1500, 1300},
//		[]float64{20, 20, 20},
//		[]float64{1, 0.5, 0},
//	)
//
// # License
//
// MIT license, see LICENSE for info.
package elo

import (
	"errors"
	"math"
	"slices"
	"time"
)

// The rating system as described by Arpad Elo.
// Has a 400-point spread, meaning someone with
// a rating 400 points higher will be 10x more
// likely to win.
var Default = Elo{400}

// A representation of a player with a rating, for use with the helper functions.
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
	// The rating difference between 2 players which would give one player 10-to-1 odds of winning.
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
	// keeping a list of indices to reverse the process.
	// this might be the most confusing part of the whole
	// process. It's generating a mapping from ratings => r
	// based on score.
	idx := make([]int, l)
	for i := range l {
		idx[i] = i
	}
	slices.SortStableFunc(idx, cmp(normalizedScores))

	// use the index lookup to create new slices in score order.
	// These will have to be put back into input order after the
	// calculations are done.
	r, k, s := make([]float64, l), make([]float64, l), make([]float64, l)
	for i, j := range idx {
		r[i] = ratings[j]
		k[i] = kFactors[j]
		s[i] = normalizedScores[j]
	}

	// do the actual Elo calculations
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

// Calculate the new Elo ratings for a match with a 1st place, 2nd place, etc.
// The player[0] is 1st place, player[1] is 2nd, and so on.
func (e Elo) Place(players []Player) ([]float64, error) {
	scores := make([]float64, len(players))
	lerp(scores)
	return e.FFA(players, scores)
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

// Mutate the elements of s so that all values are from [0..1] while
// keeping their relative proportion of magnitudes intact.
func normalize(s []float64) {
	// don't bother normalizing less than 2 - it won't affect the calculation.
	// Calculate will short-circuit these.
	if len(s) < 2 {
		return
	}

	// normalize ties (when min==max) to 0.5.
	// this way we don't get nonsense answers from ties.
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

// compares two entries in a slice s give two indices to s
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

// fills a slice with the linear interpolation from 1..0.
func lerp(s []float64) {
	// nothing to lerp
	if len(s) == 0 {
		return
	}
	// one value interpolated between 0 and 1 is 1/2.
	if len(s) == 1 {
		s[0] = 0.5
		return
	}
	// divide by zero prevented by len(s) >= 2
	step := 1 / float64(len(s)-1)
	for i := range s {
		s[i] = 1 - float64(i)*step
	}
}
