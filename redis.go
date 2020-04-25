package conf

import (
	"errors"
	"time"

	"github.com/go-redis/redis/v7"
)

//RedisConfig redis配置
type RedisConfig struct {
	Addr         string        `yaml:"addr"` //127.0.0.1:6379
	Addrs        []string      `yaml:"addrs"`
	Passwd       string        `yaml:"password"`
	DialTimeout  time.Duration `yaml:"dialtimeout"`
	ReadTimeout  time.Duration `yaml:"readtimeout"`
	WriteTimeout time.Duration `yaml:"writetimeout"`
	PoolSize     int           `yaml:"poolsize"`
	PoolTimeout  time.Duration `yaml:"pooltimeout"`
	Cluster      bool          `yaml:"cluster"`
}

func loadConfig(cfg *RedisConfig) (*RedisConfig, error) {
	if len(cfg.Addrs) == 0 && cfg.Addr == "" {
		return nil, errors.New("miss redis address(addr), cluster(addrs) setting")
	}
	// 如果多个IP强制认为是cluster
	if len(cfg.Addrs) > 0 {
		cfg.Cluster = true
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 2 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 10
	}
	if cfg.PoolTimeout == 0 {
		cfg.PoolTimeout = 20 * time.Second
	}
	return cfg, nil
}

func RedisClient(config *RedisConfig) (redis.Cmdable, error) {
	cfg, err := loadConfig(config)
	if err != nil {
		return nil, err
	}
	var client redis.Cmdable
	//single client
	if !cfg.Cluster {
		option := redis.Options{
			Addr:         cfg.Addr,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
			PoolTimeout:  cfg.PoolTimeout,
			Password:     cfg.Passwd,
			DB:           0,
		}
		client = redis.NewClient(&option)
	} else {
		option := redis.ClusterOptions{
			Addrs:        cfg.Addrs,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
			PoolTimeout:  cfg.PoolTimeout,
			Password:     cfg.Passwd,
		}
		client = redis.NewClusterClient(&option)
	}
	if client == nil {
		return nil, errors.New("init redis client error")
	}
	return client, nil
}
