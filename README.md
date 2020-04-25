# conf

提取读取配置的功能为基础库

## 基本使用方法

定义了几个标准格式的配置结构
http: 定义访问域名和端口 例如":80" , "127.0.0.1:8080"
mysql: 定义mysql访问的配置内容
log: 定义level(默认info）和存储文件地址（空为stdout)

## 文件格式 yaml

一般为各个基础组件的组合模式

```yaml
mysql:
  user: root
  password: password
  database: dbname
  addr: "127.0.0.1:3306"
  collection: "utf8mb4_unicode_ci"
  params:
    parseTime: "true"
http:
  listen: ":3333"
log:
  level: "info"
  file: "/tmp/log":
```

## 配置读取

```golang
type config struct {
    MysqlConfig conf.MysqlConfig      `yaml:"mysql"`
    HTTPConfig  conf.HTTPConfig       `yaml:"http"`
    LogConfig   conf.Log              `yaml:"log"`
}
var c config
var configFilePath = "conf.yml"
conf.LoadConfig(configFilePath, &c)
```

## 基础组件

### mysql

```golang
// mysqlConfig 是 *conf.MysqlConfig 对象
gormDB, err := conf.MySQLClient(mysqlConfig)

```

### redis

```golang
// config 是 *conf.RedisConfig  
redisClient, err := RedisClient(config)
//redisClient = redis.Cmable 支持single/cluster
```

### mongodb

```golang
// config 是 *conf.MongoDBConfig  
mongo, err := MongoDBClient(config)
```

### rabbitMQ

```golang
//参考amqp/README.md
client := RabbitMQClientWithLog(conf, logger, false)
```

### log

```golang
//基础方法
logger := conf.Logger(conf)
```
