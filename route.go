package llmrouter

// Route represents one selectable combination of provider, model, and
// API key.
type Route struct {
	ID       string
	Provider string
	Model    string
	APIKey   string

	// BaseURL overrides the default endpoint for Provider. Leave it empty
	// to let the application use Provider's usual endpoint; set it to
	// point at a self-hosted, proxied, or otherwise custom OpenAI-compatible
	// endpoint. The router treats it as opaque metadata, same as APIKey.
	BaseURL string

	Weight int
}
