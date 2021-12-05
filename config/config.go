package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var GlobalConfig *Config

type Config struct {
	StoreType     string        `yaml:"store_type" json:"store_type"`
	Host          string        `yaml:"server_host" json:"server_host"`
	Port          int           `yaml:"server_port" json:"server_port"`
	LocalDBConfig *BoltDBConfig `yaml:"local_config" json:"local_config"`
	ConsulConfig  *ConsulConfig `yaml:"consul_config" json:"consul_config"`
}

func NewConfig(configPath string) (*Config, error) {
	c := &Config{}
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not read config: %s", err))
	}

	err = yaml.Unmarshal([]byte(b), &c)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not unmarshal config: %v", err))
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Serialize() (string, error) {
	if b, err := yaml.Marshal(c); err != nil {
		return "", err
	} else {
		return string(b), nil
	}
}

func (c *Config) validate() error {
	validStoreType := map[string]bool{"local": true, "consul": true}
	if ok, _ := validStoreType[c.StoreType]; !ok {
		return errors.New("Only the local data store is currently supported")
	}

	return nil
}
