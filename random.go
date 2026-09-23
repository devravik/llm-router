package llmrouter

import "math/rand/v2"

type randomStrategy struct{}

// Random returns a Strategy that selects an eligible route uniformly at
// random using math/rand/v2, whose top-level functions are safe for
// concurrent use without explicit locking.
func Random() Strategy {
	return randomStrategy{}
}

func (randomStrategy) Next(routes []Route) (Route, error) {
	if len(routes) == 0 {
		return Route{}, ErrNoRoutes
	}
	return routes[rand.IntN(len(routes))], nil
}
