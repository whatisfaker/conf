package amqp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/whatisfaker/zaptrace/tracing"

	"github.com/opentracing/opentracing-go/ext"

	"github.com/opentracing-contrib/go-amqp/amqptracer"

	"github.com/opentracing/opentracing-go"

	"github.com/streadway/amqp"
	"github.com/whatisfaker/zaptrace/log"
	"go.uber.org/zap"
)

var AMQPClosedErr = errors.New("RabbitMQ Consume Closed")

type rabbitMQ struct {
	connURL string
	conn    *amqp.Connection
	log     *log.Factory
	tracing bool
	mutex   sync.Mutex
}

//NewRabbitMQClientByAMQPURI URI refer to https://www.rabbitmq.com/uri-spec.html
func NewRabbitMQClientByAMQPURI(amqpURL string, log *log.Factory, tracing bool) Client {
	a := new(rabbitMQ)
	a.connURL = amqpURL
	a.log = log
	a.tracing = tracing
	return a
}

func NewSimpleRabbitMQClientByAMQPURI(amqpURL string) Client {
	return NewRabbitMQClientByAMQPURI(amqpURL, log.NewStdLogger("info"), false)
}

func NewRabbitMQClient(addr string, user string, password string, log *log.Factory, tracing bool) Client {
	if user == "" {
		user = "guest"
		password = "guest"
	}
	a := new(rabbitMQ)
	a.connURL = fmt.Sprintf("amqp://%s:%s@%s/", user, password, addr)
	a.log = log
	a.tracing = tracing
	return a
}

func (c *rabbitMQ) Logger() *log.Factory {
	return c.log
}

func (c *rabbitMQ) getConn() (*amqp.Connection, error) {
	var err error
	if c.conn == nil || c.conn.IsClosed() {
		c.conn, err = amqp.Dial(c.connURL)
		if err != nil {
			return nil, err
		}
	}
	return c.conn, nil
}

func (c *rabbitMQ) Pub(ctx context.Context, exchange string, data []byte, compressed ...bool) error {
	msg := amqp.Publishing{
		Headers:     amqp.Table{},
		ContentType: "application/json",
		Body:        data,
	}
	if len(compressed) > 0 && compressed[0] {
		msg.ContentEncoding = "gzip"
		msg.Body = gzipCompress(msg.Body)
	}
	if c.tracing {
		var sp opentracing.Span
		ctx, sp = tracing.QuickStartSpan(ctx, "AMQP Produce", ext.SpanKindProducer)
		defer sp.Finish()
		//ignore inject error
		_ = amqptracer.Inject(sp, msg.Headers)
	}
	c.log.Trace(ctx).Debug("[Pub/Sub] Publish Message", zap.String("exchange", exchange), zap.String("data", string(data)))
	conn, err := c.getConn()
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Publish Message Error(Dial)", zap.String("exchange", exchange), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	//channel加锁和释放（1 conn = 2048 channel)
	c.mutex.Lock()
	ch, err := conn.Channel()
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Publish Message Error(Channel)", zap.String("exchange", exchange), zap.String("data", string(data)), zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	defer ch.Close()
	c.mutex.Unlock()
	err = ch.ExchangeDeclare(exchange, "fanout", true, false, false, false, nil)
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Publish Message Error(ExchangeDeclare)", zap.String("exchange", exchange), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	err = ch.Publish(exchange, "", false, false, msg)
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Publish Message Error(Publish)", zap.String("exchange", exchange), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	return nil
}

func (c *rabbitMQ) Sub(ctx context.Context, exchange string, callback func(context.Context, []byte) error) error {
	conn, err := c.getConn()
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Subscribe Message Error(Dial)", zap.String("exchange", exchange), zap.Error(err))
		return err
	}
	//channel加锁和释放（1 conn = 2048 channel)
	c.mutex.Lock()
	ch, err := conn.Channel()
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Subscribe Message Error(Channel)", zap.String("exchange", exchange), zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	c.mutex.Unlock()
	err = ch.ExchangeDeclare(exchange, "fanout", true, false, false, false, nil)
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Subscribe Message Error(ExchangeDeclare)", zap.String("exchange", exchange), zap.Error(err))
		return err
	}

	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Subscribe Message Error(QueueDeclare)", zap.String("exchange", exchange), zap.Error(err))
		return err
	}

	err = ch.QueueBind(
		q.Name,   // queue name
		"",       // routing key
		exchange, // exchange
		false,
		nil)
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Subscribe Message Error(QueueBind)", zap.String("exchange", exchange), zap.Error(err))
		return err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		c.log.Trace(ctx).Error("[Pub/Sub] Subscribe Message Error(Consume)", zap.String("exchange", exchange), zap.Error(err))
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				c.log.Trace(ctx).Error("[Pub/Sub] Consume msg closed")
				return AMQPClosedErr
			}
			err := func() error {
				if c.tracing {
					spCtx, _ := amqptracer.Extract(d.Headers)
					sp := opentracing.StartSpan(
						"AMQP consume",
						opentracing.FollowsFrom(spCtx),
					)
					defer sp.Finish()
					ctx = opentracing.ContextWithSpan(ctx, sp)
				}
				if d.ContentEncoding == "gzip" {
					d.Body, err = gzipUncompress(d.Body)
					if err != nil {
						c.log.Trace(ctx).Error("[Pub/Sub] Consumne Message(Uncompress Error)", zap.Error(err))
					}
				}
				c.log.Trace(ctx).Debug("[Pub/Sub] Consumne Message", zap.String("exchange", exchange), zap.String("data", string(d.Body)))
				return callback(ctx, d.Body)
			}()
			if err != nil {
				return err
			}
		}
	}
}

func (c *rabbitMQ) CloseConn() {
	if c.conn != nil && !c.conn.IsClosed() {
		c.conn.Close()
	}
}

func (c *rabbitMQ) RoutePub(ctx context.Context, exchange string, route string, data []byte, compressed ...bool) error {
	msg := amqp.Publishing{
		Headers:     amqp.Table{},
		ContentType: "application/json",
		Body:        data,
	}
	if len(compressed) > 0 && compressed[0] {
		msg.ContentEncoding = "gzip"
		msg.Body = gzipCompress(msg.Body)
	}
	if c.tracing {
		var sp opentracing.Span
		ctx, sp = tracing.QuickStartSpan(ctx, "AMQP Produce", ext.SpanKindProducer)
		defer sp.Finish()
		//ignore inject error
		_ = amqptracer.Inject(sp, msg.Headers)
	}
	c.log.Trace(ctx).Debug("[RoutePub/Sub] Publish Message", zap.String("exchange", exchange), zap.String("route", route), zap.String("data", string(data)))
	conn, err := c.getConn()
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Publish Message(Dial)", zap.String("exchange", exchange), zap.String("route", route), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	//channel加锁和释放（1 conn = 2048 channel)
	c.mutex.Lock()
	ch, err := conn.Channel()
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Publish Message(Channel)", zap.String("exchange", exchange), zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	defer ch.Close()
	c.mutex.Unlock()
	err = ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Publish Message(ExchangeDeclare)", zap.String("exchange", exchange), zap.String("route", route), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	err = ch.Publish(exchange, route, false, false, msg)
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Publish Message(Publish)", zap.String("exchange", exchange), zap.String("route", route), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	return nil
}

func (c *rabbitMQ) RouteSub(ctx context.Context, exchange string, route string, callback func(context.Context, []byte) error) error {
	conn, err := c.getConn()
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Subscribe Message(Dial)", zap.String("exchange", exchange), zap.String("route", route), zap.Error(err))
		return err
	}
	//channel加锁和释放（1 conn = 2048 channel)
	c.mutex.Lock()
	ch, err := conn.Channel()
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Subscribe Message(Channel)", zap.String("exchange", exchange), zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	c.mutex.Unlock()
	err = ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Subscribe Message(ExchangeDeclare)", zap.String("exchange", exchange), zap.String("route", route), zap.Error(err))
		return err
	}

	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Subscribe Message(QueueDeclare)", zap.String("exchange", exchange), zap.String("route", route), zap.Error(err))
		return err
	}

	err = ch.QueueBind(
		q.Name,   // queue name
		route,    // routing key
		exchange, // exchange
		false,
		nil)
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Subscribe Message(QueueBind)", zap.String("exchange", exchange), zap.String("route", route), zap.Error(err))
		return err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		c.log.Trace(ctx).Error("[RoutePub/Sub] Subscribe Message(Consume)", zap.String("exchange", exchange), zap.String("route", route), zap.Error(err))
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				c.log.Trace(ctx).Error("[RoutePub/Sub] Consume msg closed")
				return AMQPClosedErr
			}
			err := func() error {
				if c.tracing {
					spCtx, _ := amqptracer.Extract(d.Headers)
					sp := opentracing.StartSpan(
						"AMQP consume",
						opentracing.FollowsFrom(spCtx),
					)
					defer sp.Finish()
					ctx = opentracing.ContextWithSpan(ctx, sp)
				}
				if d.ContentEncoding == "gzip" {
					d.Body, err = gzipUncompress(d.Body)
					if err != nil {
						c.log.Trace(ctx).Error("[RoutePub/Sub] Consumne Message(Uncompress Error)", zap.Error(err))
					}
				}
				c.log.Trace(ctx).Debug("[RoutePub/Sub] Consumne Message", zap.String("exchange", exchange), zap.String("route", route), zap.String("data", string(d.Body)))
				return callback(ctx, d.Body)
			}()
			if err != nil {
				return err
			}
		}
	}
}

func (c *rabbitMQ) Produce(ctx context.Context, queueName string, data []byte, compressed ...bool) error {
	msg := amqp.Publishing{
		Headers:     amqp.Table{},
		ContentType: "application/json",
		Body:        data,
	}
	if len(compressed) > 0 && compressed[0] {
		msg.ContentEncoding = "gzip"
		msg.Body = gzipCompress(msg.Body)
	}
	if c.tracing {
		var sp opentracing.Span
		ctx, sp = tracing.QuickStartSpan(ctx, "AMQP Produce", ext.SpanKindProducer)
		defer sp.Finish()
		//ignore inject error
		_ = amqptracer.Inject(sp, msg.Headers)
	}
	c.log.Trace(ctx).Debug("[Produce/Consume] Produce Message", zap.String("queue", queueName), zap.String("data", string(data)))
	conn, err := c.getConn()
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Produce Message(Dial)", zap.String("queue", queueName), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	c.mutex.Lock()
	ch, err := conn.Channel()
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Produce Message(Channel)", zap.String("queue", queueName), zap.String("data", string(data)), zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	defer ch.Close()
	c.mutex.Unlock()
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Produce Message(QueueDeclare)", zap.String("queue", queueName), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	err = ch.Publish("", q.Name, false, false, msg)
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Produce Message(Publish)", zap.String("queue", queueName), zap.String("data", string(data)), zap.Error(err))
		return err
	}
	return nil
}

func (c *rabbitMQ) Consume(ctx context.Context, queueName string, callback func(context.Context, []byte) error) error {
	conn, err := c.getConn()
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Consume Message(Dial)", zap.String("queue", queueName), zap.Error(err))
		return err
	}
	//channel加锁和释放（1 conn = 2048 channel)
	c.mutex.Lock()
	ch, err := conn.Channel()
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Consume Message(Channel)", zap.String("queue", queueName), zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	c.mutex.Unlock()
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Consume Message(QueueDeclare)", zap.String("queue", queueName), zap.Error(err))
		return err
	}
	err = ch.Qos(
		1,     // prefetch count 每次取一个
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Consume Message(Qos)", zap.String("queue", queueName), zap.Error(err))
		return err
	}
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		c.log.Trace(ctx).Error("[Produce/Consume] Consume Message(Consume)", zap.String("queue", queueName), zap.Error(err))
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				c.log.Trace(ctx).Error("[Produce/Consume] Consume msg closed")
				return AMQPClosedErr
			}
			err := func() error {
				if c.tracing {
					spCtx, _ := amqptracer.Extract(d.Headers)
					//if err != nil {
					//mute warning
					//c.log.Trace(ctx).Warn("[Produce/Consume] Consume Message(Tracing Extract)", zap.String("queue", queueName), zap.Error(err))
					//}
					sp := opentracing.StartSpan(
						"AMQP Consume",
						opentracing.FollowsFrom(spCtx),
					)
					defer sp.Finish()
					ext.SpanKindConsumer.Set(sp)
					ctx = opentracing.ContextWithSpan(ctx, sp)
				}
				if d.ContentEncoding == "gzip" {
					d.Body, err = gzipUncompress(d.Body)
					if err != nil {
						c.log.Trace(ctx).Error("[Produce/Consume] Consumne Message(Uncompress Error)", zap.Error(err))
					}
				}
				c.log.Trace(ctx).Debug("[Produce/Consume] Consume Message", zap.String("queue", queueName), zap.String("data", string(d.Body)))
				return callback(ctx, d.Body)
			}()
			if err != nil {
				return err
			}
		}
	}
}
