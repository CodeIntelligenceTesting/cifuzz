package config

type Config struct {
	*ProjectConfig
	ProjectDir string
}

func NewConfig() *Config {
	return &Config{
		ProjectConfig: &ProjectConfig{},
	}
}
