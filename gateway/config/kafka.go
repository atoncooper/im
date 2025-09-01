package config

import (
	"errors"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

var (
	producer  *kafka.Writer
	consumer  *kafka.Reader
	dlqWriter *kafka.Writer
	kafkaOnce sync.Once
)

type KafkaConf struct {
	Brokers      []string
	Topic        string
	RequiredAcks int
	MaxRetry     int
	GroupId      string
	DLQTopic     string
}

func NewKafak(conf *KafkaConf) error {

	var err error
	kafkaOnce.Do(func() {
		producer = kafka.NewWriter(kafka.WriterConfig{
			Brokers:      conf.Brokers,
			Topic:        conf.Topic,
			RequiredAcks: conf.RequiredAcks,
			MaxAttempts:  conf.MaxRetry,
			BatchTimeout: 10 * time.Millisecond,
		})

		consumer = kafka.NewReader(kafka.ReaderConfig{
			Brokers:     conf.Brokers,
			Topic:       conf.Topic,
			GroupID:     conf.GroupId,
			MaxWait:     30 * time.Second,
			StartOffset: kafka.FirstOffset,
		})

		if conf.DLQTopic != "" {
			// 初始化死信队列
			topic := conf.Topic + conf.DLQTopic
			dlqWriter = kafka.NewWriter(kafka.WriterConfig{
				Brokers:      conf.Brokers,
				Topic:        topic,
				RequiredAcks: conf.RequiredAcks,
				MaxAttempts:  conf.MaxRetry,
				BatchTimeout: 10 * time.Millisecond,
			})
		}
	})

	return err
}

func KafkaProducerTemplate() *kafka.Writer {
	if producer == nil {
		panic(errors.New("kafka producer is nil"))
	}
	return producer
}

func KafkaConsumerTemplate() *kafka.Reader {
	if consumer == nil {
		panic(errors.New("kafka consumer is nil"))
	}
	return consumer
}

func KafkaDLQTemplate() *kafka.Writer {
	if dlqWriter == nil {
		panic(errors.New("kafka dlq is nil"))
	}
	return dlqWriter
}
