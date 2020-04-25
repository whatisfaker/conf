# AMQP 消息队列

## RabbitMQ实现

### 使用方法

生产/消费模式(其他模式参考接口注释)

生产

```golang
    amqpURL := os.Getenv("AMQP_URI")
    timer := time.NewTicker(5 * time.Second)
    ctx := context.Background()
    c := amqp.NewRabbitMQClientByAMQPURI(amqpURL, logger.With(zap.String("component", "producer")), true)
    //默认不压缩
    err := c.Produce(ctx, "topic", []byte("{\"code\": 100 }"))
    //压缩传输数据
    err := c.Produce(ctx, "topic", []byte("{\"code\": 100 }"), true)
```

消费

```golang
    amqpURL := os.Getenv("AMQP_URI")
    ctx := context.Background()
    c := amqp.NewRabbitMQClientByAMQPURI(amqpURL, logger.With(zap.String("component", "producer")), true)
    //这是一个阻塞函数, 一般协程启动 或errgroup启动
    err = c.Consume(ctx, "topic", func(ctx context.Context, data []byte) error {
        //回调函数 return 非nil会退出阻塞函数，具体情况根据业务而定，如果不想退出，就包掉error 返回nil.
        return nil
    })
```

特殊说明

```golang
    //此库是复用conn, 保持长链, 如果需要手动关闭请使用此方法
    c.CloseConn()
```
