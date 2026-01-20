// Package ha provides High Availability coordination for multi-region deployments.
// In OSS, this is a stub. In enterprise, it provides:
// - Health check coordination
// - Leader election
// - Request routing across regions
package ha

// Config holds HA configuration options.
type Config struct {
	// Enabled enables HA coordination (enterprise only)
	Enabled bool `yaml:"enabled,omitempty"`
	// Region is the deployment region identifier
	Region string `yaml:"region,omitempty"`
	// CoordinatorURL is the HA coordinator service URL
	CoordinatorURL string `yaml:"coordinatorUrl,omitempty"`
}
