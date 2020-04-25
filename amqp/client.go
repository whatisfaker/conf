package amqp

import (
	"context"

	"github.com/whatisfaker/zaptrace/log"
)

//Client AMQPClient
type Client interface {
	//Pub 发布订阅模式（发布） exchange
	Pub(context.Context, string, []byte, ...bool) error
	//Sub 发布订阅模式BLOCK函数（订阅，绑定的所有Channel同时都到消息）
	Sub(context.Context, string, func(context.Context, []byte) error) error
	//RoutePub 路由发布订阅模式 （在一个exchange上，使用某个routeKey发布消息）
	RoutePub(context.Context, string, string, []byte, ...bool) error
	//RouteSub 路由发布订阅模式BLOCK函数 （在一个exchange上，匹配某个routeKey订阅消费，绑定同一个routeKey的Channel同时收到消息）
	RouteSub(context.Context, string, string, func(context.Context, []byte) error) error
	//Produce 生成消费模式
	Produce(context.Context, string, []byte, ...bool) error
	//Consume 生产消费模式BLOCK函数（同时绑定的Channel，按照round robin消费，每个channel消费的个数可配，现在默认是1，例如 2个消费者，顺序消费1234，结果为，1，3 和 2，4）
	Consume(context.Context, string, func(context.Context, []byte) error) error
	//CloseConn 手动关闭链接（特殊情况不保持链接）
	CloseConn()
	//Logger 获取内部用的logger
	Logger() *log.Factory
}
