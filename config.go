package main

type Config struct {
	GCPConfig *GCPConfig
}

type GCPConfig struct {
	ProjectID   string
	SourceImage string
}

func EmptyConfig() *Config {
	return &Config{
		GCPConfig: &GCPConfig{
			ProjectID:   "",
			SourceImage: "",
		},
	}
}
