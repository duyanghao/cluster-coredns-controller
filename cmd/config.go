package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type coreDnsCfg struct {
	CorefilePath         string `yaml:"corefilePath,omitempty"`
	ZonesDir             string `yaml:"zonesDir,omitempty"`
	WildcardDomainSuffix string `yaml:"wildcardDomainSuffix,omitempty"`
	Interval             string `yaml:"interval,omitempty"`
	Jitter               string `yaml:"jitter,omitempty"`
}

type clusterServerCfg struct {
	Addr string `yaml:"addr,omitempty"`
	Port int    `yaml:"port,omitempty"`
	URI  string `yaml:"uri,omitempty"`
}

type daemonCfg struct {
	MaxIdleConns        int `yaml:"maxIdleConns,omitempty"`
	MaxIdleConnsPerHost int `yaml:"maxIdleConnsPerHost,omitempty"`
	IdleConnTimeout     int `yaml:"idleConnTimeout,omitempty"`
	SyncInterval        int `yaml:"syncInterval,omitempty"`
}

type Config struct {
	ClusterServerCfg *clusterServerCfg `yaml:"clusterServerCfg,omitempty"`
	DaemonCfg        *daemonCfg        `yaml:"daemonCfg,omitempty"`
	CoreDnsCfg       *coreDnsCfg       `yaml:"coreDnsCfg,omitempty"`
}

// validate the configuration
func (c *Config) validate() error {
	if c.DaemonCfg.MaxIdleConns <= 0 || c.DaemonCfg.MaxIdleConnsPerHost <= 0 || c.DaemonCfg.IdleConnTimeout <= 0 || c.DaemonCfg.SyncInterval <= 0 {
		return fmt.Errorf("invalid daemon configurations, please check ...")
	}
	if c.ClusterServerCfg.Addr == "" || c.ClusterServerCfg.URI == "" || c.ClusterServerCfg.Port <= 0 {
		return fmt.Errorf("invalid cluster server configurations, please check ...")
	}
	if c.CoreDnsCfg.CorefilePath == "" || c.CoreDnsCfg.ZonesDir == "" {
		return fmt.Errorf("invalid coredns path configurations, please check ...")
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
		return nil, fmt.Errorf("Failed to read configuration file: %s,error: %s", path, err)
	}
	if err = yaml.Unmarshal(contents, c); err != nil {
		return nil, fmt.Errorf("Failed to parse configuration,error: %s", err)
	}
	if err = c.validate(); err != nil {
		return nil, fmt.Errorf("Invalid configuration,error: %s", err)
	}
	return c, nil
}
