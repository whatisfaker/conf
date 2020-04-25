package conf

import (
	"github.com/whatisfaker/conf/amqp"
	"github.com/whatisfaker/zaptrace/log"
)

type RabbitMQConfig struct {
	URI      string `yaml:"uri"`
	Address  string `yaml:"addr"`
	Username string `yaml:"user"`
	Password string `yaml:"password"`
}

func RabbitMQClientWithLog(conf *RabbitMQConfig, log *log.Factory, trace bool) amqp.Client {
	if conf.URI != "" {
		return amqp.NewRabbitMQClientByAMQPURI(conf.URI, log, trace)
	}
	return amqp.NewRabbitMQClient(conf.Address, conf.Username, conf.Password, log, trace)
}
