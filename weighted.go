package llmrouter

import (
	"fmt"
	"math/rand/v2"
)

type weightedStrategy struct{}

// Weighted returns a Strategy that selects an eligible route with
// probability proportional to its Weight. Routes with a zero Weight are
// never selected. If every eligible route has a zero Weight, Next
// returns ErrNoRoutes.
//
// Route.Weight defaults to 0; set it explicitly on every route when using
// Weighted, or every route will be excluded.
func Weighted() Strategy {
	return weightedStrategy{}
}

func (weightedStrategy) Next(routes []Route) (Route, error) {
	if len(routes) == 0 {
		return Route{}, ErrNoRoutes
	}

	total := 0
	for _, route := range routes {
		total += route.Weight
	}
	if total <= 0 {
		return Route{}, fmt.Errorf("%w: all eligible routes have zero weight", ErrNoRoutes)
	}

	n := rand.IntN(total)
	for _, route := range routes {
		if n < route.Weight {
			return route, nil
		}
		n -= route.Weight
	}

	// Unreachable: n is always less than total, so the loop above
	// always returns before routes is exhausted.
	return Route{}, ErrNoRoutes
}
