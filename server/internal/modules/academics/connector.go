package academics

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed fixtures/*.json
var fixtureFS embed.FS

type Connector interface {
	HealthCheck(ctx context.Context) error
	FetchTerms(ctx context.Context) ([]ImportTerm, error)
	FetchCourses(ctx context.Context) ([]ImportCourse, error)
	FetchTeachers(ctx context.Context) ([]ImportTeacher, error)
	FetchOfferings(ctx context.Context) ([]ImportOffering, error)
	FetchMemberships(ctx context.Context) ([]ImportMembership, error)
}

type ConnectorFactory func(source Source) (Connector, error)

type Registry struct {
	factories map[string]ConnectorFactory
}

func NewRegistry() *Registry {
	registry := &Registry{factories: map[string]ConnectorFactory{}}
	registry.Register("fixture", newFixtureConnector)
	return registry
}

func (r *Registry) Register(provider string, factory ConnectorFactory) {
	if provider == "" || factory == nil {
		return
	}
	r.factories[provider] = factory
}

func (r *Registry) Build(source Source) (Connector, error) {
	factory, ok := r.factories[source.Provider]
	if !ok {
		return nil, fmt.Errorf("connector provider %q not registered", source.Provider)
	}
	return factory(source)
}

type fixtureConfig struct {
	Fixture string `json:"fixture"`
}

type fixtureConnector struct {
	snapshot Snapshot
}

func newFixtureConnector(source Source) (Connector, error) {
	cfg := fixtureConfig{Fixture: "buaa-default"}
	if len(source.Config) > 0 {
		if err := json.Unmarshal(source.Config, &cfg); err != nil {
			return nil, fmt.Errorf("decode fixture config: %w", err)
		}
	}
	raw, err := fixtureFS.ReadFile("fixtures/" + cfg.Fixture + ".json")
	if err != nil {
		return nil, fmt.Errorf("read fixture %q: %w", cfg.Fixture, err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, fmt.Errorf("decode fixture %q: %w", cfg.Fixture, err)
	}
	return &fixtureConnector{snapshot: snapshot}, nil
}

func (c *fixtureConnector) HealthCheck(context.Context) error { return nil }

func (c *fixtureConnector) FetchTerms(context.Context) ([]ImportTerm, error) {
	return append([]ImportTerm(nil), c.snapshot.Terms...), nil
}

func (c *fixtureConnector) FetchCourses(context.Context) ([]ImportCourse, error) {
	return append([]ImportCourse(nil), c.snapshot.Courses...), nil
}

func (c *fixtureConnector) FetchTeachers(context.Context) ([]ImportTeacher, error) {
	return append([]ImportTeacher(nil), c.snapshot.Teachers...), nil
}

func (c *fixtureConnector) FetchOfferings(context.Context) ([]ImportOffering, error) {
	return append([]ImportOffering(nil), c.snapshot.Offerings...), nil
}

func (c *fixtureConnector) FetchMemberships(context.Context) ([]ImportMembership, error) {
	return append([]ImportMembership(nil), c.snapshot.Memberships...), nil
}
