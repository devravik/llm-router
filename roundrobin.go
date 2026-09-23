package llmrouter

import "sync/atomic"

type roundRobin struct {
	counter atomic.Uint64
}

// RoundRobin returns a Strategy that selects eligible routes in
// sequential order using an internal counter. Each call to RoundRobin
// returns a fresh Strategy, so separate routers never share rotation
// state.
func RoundRobin() Strategy {
	return &roundRobin{}
}

func (s *roundRobin) Next(routes []Route) (Route, error) {
	if len(routes) == 0 {
		return Route{}, ErrNoRoutes
	}

	n := s.counter.Add(1) - 1
	return routes[n%uint64(len(routes))], nil
}
