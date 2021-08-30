package config

type ConsulConfig struct {
	Host       string `json:"host" yaml:"host"`
	DC         string `json:"dc" yaml:"dc"`
	AllowStale bool   `json:"allow_stale" yaml:"allow_stale"`
}
