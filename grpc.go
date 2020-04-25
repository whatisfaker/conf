package conf

type GRPCServerConfig struct {
	Listen string `yaml:"listen"`
}

type GRPCClientConfig struct {
	Mapping map[string]string `yaml:"mapping"`
}
