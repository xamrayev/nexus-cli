package core

// Registry holds registered providers and indexes their models.
type Registry struct {
	providers map[string]Provider
	order     []string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider to the registry. Registering the same name twice
// replaces the previous provider while keeping the original registration order.
func (r *Registry) Register(p Provider) {
	if _, exists := r.providers[p.Name()]; !exists {
		r.order = append(r.order, p.Name())
	}
	r.providers[p.Name()] = p
}

// Provider returns the named provider and whether it is registered.
func (r *Registry) Provider(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Providers returns all registered providers in registration order.
func (r *Registry) Providers() []Provider {
	out := make([]Provider, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.providers[name])
	}
	return out
}

// AllModels returns every model offered by every registered provider, with the
// Provider field tagged on each entry.
func (r *Registry) AllModels() []ModelInfo {
	var out []ModelInfo
	for _, p := range r.Providers() {
		for _, m := range p.Models() {
			mi := m
			mi.Provider = p.Name()
			out = append(out, mi)
		}
	}
	return out
}

// FindModel locates a model by id across providers. The first provider (in
// registration order) that offers the model wins. The returned ModelInfo has
// its Provider field set to the provider that serves it.
func (r *Registry) FindModel(id string) (Provider, ModelInfo, bool) {
	for _, p := range r.Providers() {
		for _, m := range p.Models() {
			if m.ID == id {
				mi := m
				mi.Provider = p.Name()
				return p, mi, true
			}
		}
	}
	return nil, ModelInfo{}, false
}
