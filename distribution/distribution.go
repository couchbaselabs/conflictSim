package distribution

import (
	"fmt"
	"math/rand"
)

type Distribution struct {
	gap        float64
	randomFunc func() float64
}

type randomType int

const (
	NotRandom randomType = iota
	PseudoRandom
	NormRandom
	ExpRandom
)

func RandomTypeFromStr(s string) (r randomType) {
	switch s {
	case "step":
		r = NotRandom
	case "exprandom":
		r = ExpRandom
	case "normrandom":
		r = NormRandom
	case "pseudorandom":
		r = PseudoRandom
	default:
		panic(fmt.Sprintf("undefined randType %v", s))
	}

	return
}

func NewDistribution(gap float64, randType randomType) Distribution {
	d := Distribution{
		gap: gap,
	}

	switch randType {
	case NotRandom:
		d.randomFunc = func() float64 {
			return 1.0
		}
	case PseudoRandom:
		d.randomFunc = rand.Float64
	case NormRandom:
		d.randomFunc = rand.NormFloat64
	case ExpRandom:
		d.randomFunc = rand.ExpFloat64
	default:
		panic(fmt.Sprintf("undefined randType %v", randType))
	}

	return d
}

func (d *Distribution) Next() float64 {
	min := 0.0
	max := d.gap
	next := min + d.randomFunc()*(max-min)
	return next
}
