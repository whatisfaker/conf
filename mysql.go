package conf

import (
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

const (
	EnvGORMDebug = "GORM_DEBUG"
)

//MysqlConfig Mysql配置结构
type MysqlConfig struct {
	//DSN 连接DSN
	DSN string `yaml:"dsn"`
	//User 数据库用户名
	User string `yaml:"user"`
	//Passwd 密码
	Passwd string `yaml:"password"`
	//Addr 地址 127.0.0.1:3306
	Addr string `yaml:"addr"`
	//Database 数据库名
	Database string `yaml:"database"`
	//Collation 字符集
	Collation string `yaml:"collation"`
	//Params 额外参数
	Params map[string]string `yaml:"params"`
	//MaxIdleConns 最大空闲连接数默认100 (maxIdle >= maxOpen to avoid time wait)
	MaxIdleConns int `yaml:"max_idle_conns"`
	//MaxOpenConns 最大连接数默认100
	MaxOpenConns int `yaml:"max_open_conns"`
	//ConnMaxLifeTime 连接最大生命周期(默认4小时) 单位秒
	ConnMaxLifeTime int `yaml:"con_max_lifetime"` //默认连接生命周期(单位秒)
	//DebugSQL
	DebugSQL bool `yaml:"debug_sql"`
}

func MySQLClient(conf *MysqlConfig) (*gorm.DB, error) {
	var dsn string
	if conf.DSN == "" {
		cfg := convertAppConfigToMySQLConfig(conf)
		dsn = cfg.FormatDSN()
	} else {
		dsn = conf.DSN
	}
	db, err := gorm.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if conf.MaxIdleConns == 0 {
		conf.MaxIdleConns = 100 //默认保持空闲的最高连接数100
	}
	if conf.MaxOpenConns == 0 {
		conf.MaxOpenConns = 100 //默认连接池最高100连接
	}
	if conf.ConnMaxLifeTime == 0 {
		conf.ConnMaxLifeTime = 14400 //默认连接生命周期 14400s = 4h
	}
	db.DB().SetMaxOpenConns(conf.MaxOpenConns)                                    // The default is 0 (unlimited)
	db.DB().SetMaxIdleConns(conf.MaxIdleConns)                                    // defaultMaxIdleConns = 2
	db.DB().SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifeTime)) // 0, connections are reused forever.
	db.LogMode(conf.DebugSQL)
	db.SingularTable(true)
	return db, nil
}

//ConvertAppConfigToMySQLConfig ...
func convertAppConfigToMySQLConfig(conf *MysqlConfig) *mysql.Config {
	config := mysql.NewConfig()
	config.User = conf.User
	config.Passwd = conf.Passwd
	config.DBName = conf.Database
	config.Addr = conf.Addr
	if conf.Collation == "" {
		config.Collation = "utf8mb4_unicode_ci"
	} else {
		config.Collation = conf.Collation
	}
	config.ParseTime = true
	config.Params = conf.Params
	return config
}
