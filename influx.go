package conf

import (
	"fmt"

	_ "github.com/influxdata/influxdb1-client"
	client "github.com/influxdata/influxdb1-client/v2"
)

type InfluxConfig struct {
	Addr     string `yaml:"addr"`
	Database string `yaml:"database"`
	UserName string `yaml:"user"`
	Password string `yaml:"password"`
}

type InfluxWrap struct {
	Client   client.Client
	Database string
}

func InfluxClient(conf *InfluxConfig) (*InfluxWrap, error) {
	client, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s", conf.Addr),
		Username: conf.UserName,
		Password: conf.Password,
	})
	if err != nil {
		return nil, err
	}
	return &InfluxWrap{
		Client:   client,
		Database: conf.Database,
	}, nil
}
