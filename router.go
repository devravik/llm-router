package llmrouter

import "sync"

// Strategy selects one route from a slice of eligible routes.
//
// Next receives only enabled routes; implementations do not need to
// filter by eligibility, handle retries, or manage cooldowns.
//
// Next may be called concurrently from multiple goroutines; implementations
// must be safe for concurrent use.
type Strategy interface {
	Next(routes []Route) (Route, error)
}

// Option configures a Router constructed by New.
type Option func(*Router)

// WithStrategy sets the strategy used to select a route on each call to
// Next. If not supplied, New defaults to RoundRobin. A nil Strategy is
// ignored, leaving the current strategy in place.
func WithStrategy(s Strategy) Option {
	return func(r *Router) {
		if s != nil {
			r.strategy = s
		}
	}
}

type routeEntry struct {
	route   Route
	enabled bool
}

// Router selects a Route from a set of configured routes according to
// its Strategy. A Router is safe for concurrent use by multiple
// goroutines.
type Router struct {
	mu       sync.RWMutex
	strategy Strategy
	entries  []*routeEntry
	index    map[string]int
}

// New creates a Router. By default it uses RoundRobin; pass WithStrategy
// to use a different strategy.
func New(opts ...Option) *Router {
	r := &Router{
		strategy: RoundRobin(),
		index:    make(map[string]int),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Add registers a route. It returns ErrInvalidRoute if the route has an
// empty ID or a negative Weight, and ErrDuplicateRoute if a route with
// the same ID is already registered.
func (r *Router) Add(route Route) error {
	if route.ID == "" || route.Weight < 0 {
		return ErrInvalidRoute
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.index[route.ID]; exists {
		return ErrDuplicateRoute
	}

	r.index[route.ID] = len(r.entries)
	r.entries = append(r.entries, &routeEntry{route: route, enabled: true})
	return nil
}

// Remove unregisters a route. It returns ErrNotFound if the ID is
// unknown.
func (r *Router) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pos, exists := r.index[id]
	if !exists {
		return ErrNotFound
	}

	r.entries = append(r.entries[:pos], r.entries[pos+1:]...)
	delete(r.index, id)
	for i := pos; i < len(r.entries); i++ {
		r.index[r.entries[i].route.ID] = i
	}
	return nil
}

// Enable marks a route eligible for selection. It returns ErrNotFound if
// the ID is unknown.
func (r *Router) Enable(id string) error {
	return r.setEnabled(id, true)
}

// Disable marks a route ineligible for selection. It returns ErrNotFound
// if the ID is unknown.
func (r *Router) Disable(id string) error {
	return r.setEnabled(id, false)
}

func (r *Router) setEnabled(id string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pos, exists := r.index[id]
	if !exists {
		return ErrNotFound
	}

	r.entries[pos].enabled = enabled
	return nil
}

// Next selects a route using the configured Strategy. It returns
// ErrNoRoutes if no route is currently eligible.
func (r *Router) Next() (Route, error) {
	r.mu.RLock()
	eligible := make([]Route, 0, len(r.entries))
	for _, e := range r.entries {
		if e.enabled {
			eligible = append(eligible, e.route)
		}
	}
	strategy := r.strategy
	r.mu.RUnlock()

	if len(eligible) == 0 {
		return Route{}, ErrNoRoutes
	}

	return strategy.Next(eligible)
}
