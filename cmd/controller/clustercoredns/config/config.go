package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type CoreDnsCfg struct {
	CorefilePath         string `yaml:"corefilePath,omitempty"`
	ZonesDir             string `yaml:"zonesDir,omitempty"`
	WildcardDomainSuffix string `yaml:"wildcardDomainSuffix,omitempty"`
	Interval             int    `yaml:"interval,omitempty"`
}

type ClusterServerCfg struct {
	// The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.
	MasterURL string `yaml:"masterURL,omitempty"`
	// Path to a kubeconfig. Only required if out-of-cluster.
	KubeConfig string `yaml:"kubeConfig,omitempty"`
	// Enable event broadcaster
	EnableEvent bool `yaml:"enableEvent,omitempty"`
}

type Config struct {
	ClusterServerCfg *ClusterServerCfg `yaml:"clusterServerCfg,omitempty"`
	CoreDnsCfg       *CoreDnsCfg       `yaml:"coreDnsCfg,omitempty"`
}

// validate the configuration
func (c *Config) validate() error {
	if c.CoreDnsCfg.CorefilePath == "" || c.CoreDnsCfg.ZonesDir == "" || c.CoreDnsCfg.WildcardDomainSuffix == "" {
		return fmt.Errorf("Invalid coredns configurations, please check ...")
	}
	if c.CoreDnsCfg.Interval <= 0 || c.CoreDnsCfg.Interval%2 != 0 {
		return fmt.Errorf("Invalid coredns reload interval parameter, please check ...")
	}
	// TODO: other configuration validate ...
	return nil
}

// LoadConfig parses configuration file and returns
// an initialized Settings object and an error object if any. For instance if it
// cannot find the configuration file it will set the returned error appropriately.
func LoadConfig(path string) (*Config, error) {
	c := &Config{}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read configuration file: %s, error: %s", path, err)
	}
	if err = yaml.Unmarshal(contents, c); err != nil {
		return nil, fmt.Errorf("Failed to parse configuration, error: %s", err)
	}
	if err = c.validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration, error: %s", err)
	}
	return c, nil
}
