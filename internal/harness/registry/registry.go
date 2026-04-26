// Package registry provides an in-memory catalog of harness adapter providers.
// Provider definitions are code-defined; the registry is not persisted to disk.
// Manifests (the persistent layer) live in internal/harness/link.
package registry

// SupportLevel indicates the degree of official support for a harness provider.
type SupportLevel string

const (
	// SupportLevelSupported marks providers that are first-party verified.
	SupportLevelSupported SupportLevel = "supported"

	// SupportLevelExperimental marks providers under active development.
	SupportLevelExperimental SupportLevel = "experimental"

	// SupportLevelManual marks providers installed and managed by the user.
	SupportLevelManual SupportLevel = "manual"
)

// Provider describes a registered harness adapter provider and its capabilities.
type Provider struct {
	// Name is the unique identifier for the provider (e.g., "hermes", "manual").
	Name string

	// SupportLevel declares the official support tier for this provider.
	SupportLevel SupportLevel

	// Capabilities is an explicit list of operation identifiers the provider
	// supports (e.g., "submit_prompt", "get_snapshot"). Never inferred from
	// directory names or file presence.
	Capabilities []string
}

// Registry is an in-memory store of Provider definitions.
// Its zero value is not usable; use New.
type Registry struct {
	providers map[string]Provider
}

// New returns an initialized, empty Registry.
func New() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds or replaces the provider definition identified by p.Name.
// Calling Register with a name that already exists silently replaces the entry.
func (r *Registry) Register(p Provider) {
	r.providers[p.Name] = p
}

// Get returns the provider registered under name and whether it was found.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered providers. The order is undefined.
func (r *Registry) List() []Provider {
	out := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		out = append(out, p)
	}
	return out
}

// SupportedProviders returns the subset of registered providers whose
// SupportLevel is SupportLevelSupported.
func (r *Registry) SupportedProviders() []Provider {
	var out []Provider
	for _, p := range r.providers {
		if p.SupportLevel == SupportLevelSupported {
			out = append(out, p)
		}
	}
	return out
}
