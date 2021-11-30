package config

type BoltDBConfig struct {
	TargetStorePath string `yaml:"store_path" json:"store_path"`
}

func newBoltDBConfig() *BoltDBConfig {
	c := &BoltDBConfig{}
	return c
}

func LoadConfig(filePath string) (*BoltDBConfig, error) {

	c := newBoltDBConfig()

	return c, nil
}

func (c *BoltDBConfig) validate() error {
	return nil
}
