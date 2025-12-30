// Package collector provides metric collection implementations.
package collector

import (
	"context"
	"fmt"
	"sync"

	"github.com/bellistech/metrics-system/internal/logger"
	"github.com/bellistech/metrics-system/pkg/metrics"
)

// Collector is the interface all metric collectors must implement.
type Collector interface {
	// Name returns the collector name (e.g., "cpu", "memory").
	Name() string
	// Collect gathers metrics and returns them.
	Collect(ctx context.Context) ([]metrics.Metric, error)
}

// CollectorConfig holds configuration passed to collector factories.
type CollectorConfig struct {
	Hostname    string
	MountPoints []string            // For disk collector
	Interfaces  []string            // For network collector
	Options     map[string]string   // Generic options for custom collectors
}

// CollectorFactory is a function that creates a new collector instance.
type CollectorFactory func(cfg CollectorConfig) Collector

// Global factory registry - collectors register themselves via init()
var (
	factoryMu   sync.RWMutex
	factories   = make(map[string]CollectorFactory)
)

// RegisterFactory registers a collector factory by name.
// This is typically called in init() functions of collector files.
func RegisterFactory(name string, factory CollectorFactory) {
	factoryMu.Lock()
	defer factoryMu.Unlock()
	if _, exists := factories[name]; exists {
		logger.Warn("Overwriting collector factory: %s", name)
	}
	factories[name] = factory
	logger.Debug("Registered collector factory: %s", name)
}

// GetFactory retrieves a collector factory by name.
func GetFactory(name string) (CollectorFactory, bool) {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// ListFactories returns all registered factory names.
func ListFactories() []string {
	factoryMu.RLock()
	defer factoryMu.RUnlock()
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	return names
}

// Registry holds registered collectors.
type Registry struct {
	mu         sync.RWMutex
	collectors map[string]Collector
}

// NewRegistry creates a new collector registry.
func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
	}
}

// Register adds a collector to the registry.
func (r *Registry) Register(c Collector) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[c.Name()] = c
}

// RegisterByName creates and registers a collector using its factory.
// This allows collectors to be registered purely by config.
func (r *Registry) RegisterByName(name string, cfg CollectorConfig) error {
	factory, ok := GetFactory(name)
	if !ok {
		return fmt.Errorf("unknown collector type: %s (available: %v)", name, ListFactories())
	}

	collector := factory(cfg)
	r.Register(collector)
	logger.Debug("Registered collector: %s", name)
	return nil
}

// RegisterFromConfig registers multiple collectors from a list of names.
// This is the main entry point for config-driven registration.
func (r *Registry) RegisterFromConfig(names []string, cfg CollectorConfig) error {
	var errs []error
	for _, name := range names {
		if err := r.RegisterByName(name, cfg); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to register some collectors: %v", errs)
	}
	return nil
}

// Get retrieves a collector by name.
func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.collectors[name]
	return c, ok
}

// List returns all registered collector names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.collectors))
	for name := range r.collectors {
		names = append(names, name)
	}
	return names
}

// CollectAll runs all registered collectors and returns combined metrics.
func (r *Registry) CollectAll(ctx context.Context) ([]metrics.Metric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allMetrics []metrics.Metric
	var errs []error

	for name, c := range r.collectors {
		m, err := c.Collect(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		allMetrics = append(allMetrics, m...)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logger.Error("Collection error: %v", err)
		}
	}

	return allMetrics, nil
}

// CollectFrom runs specific collectors and returns their metrics.
func (r *Registry) CollectFrom(ctx context.Context, names []string) ([]metrics.Metric, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allMetrics []metrics.Metric
	var errs []error

	for _, name := range names {
		c, ok := r.collectors[name]
		if !ok {
			errs = append(errs, fmt.Errorf("collector not found: %s", name))
			continue
		}

		m, err := c.Collect(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		allMetrics = append(allMetrics, m...)
	}

	if len(errs) > 0 {
		for _, err := range errs {
			logger.Error("Collection error: %v", err)
		}
	}

	return allMetrics, nil
}
