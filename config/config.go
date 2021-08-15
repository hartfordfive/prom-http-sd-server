package config

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	StoreType       string `yaml:"store_type" json:"store_type"`
	TargetStorePath string `yaml:"store_path" json:"store_path"`
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
}

func newConfig() *Config {
	c := &Config{
		StoreType: "local",
		Host:      "127.0.0.1",
		Port:      80,
	}
	return c
}

func LoadConfig(filePath string) (*Config, error) {

	c := newConfig()
	b, err := ioutil.ReadFile(filePath)
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

func (c *Config) validate() error {
	validStoreType := map[string]bool{"local": true}
	if ok, _ := validStoreType[c.StoreType]; !ok {
		return errors.New("Only the local data store is currently supported")
	}
	return nil
}
