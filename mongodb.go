package conf

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBConfig struct {
	URI            string        `yaml:"uri"`
	Database       string        `yaml:"database"`
	ContextTimeout time.Duration `yaml:"timeout"`
}

func MongoDBClient(conf *MongoDBConfig) (*mongo.Client, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(conf.URI))
	if err != nil {
		return nil, err
	}
	if conf.ContextTimeout == 0 {
		conf.ContextTimeout = 30 * time.Second
	}
	connectCtx, cancel := context.WithTimeout(context.TODO(), conf.ContextTimeout)
	defer cancel()
	// Connect to MongoDB
	err = client.Connect(connectCtx)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(context.TODO(), conf.ContextTimeout)
	defer cancel()
	// Check the connection
	err = client.Ping(pingCtx, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}
