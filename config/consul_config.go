package config

type ConsulConfig struct {
	Host string `json:"host" yaml:"host"`
	DC   string `json:"dc" yaml:"dc"`
}
