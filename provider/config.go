package provider

type Config struct {
	Backend map[string]BackendConfig `mapstructure:"backend"`
}

type BackendConfig struct {
	Csrf       string `mapstructure:"csrf"`
	CsrfHeader string `mapstructure:"csrf-header"`
	Session    string `mapstructure:"session"`
}
