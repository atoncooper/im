package tests

import (
	"context"
	"gateway/config"
	"gateway/service"
	"testing"
)

func TestKafka(t *testing.T) {
	// 初始化kafka连接
	conf := config.KafkaConf{
		Brokers:      []string{config.GatewayCfg.Application.Component.Kafka.Brokers},
		Topic:        config.GatewayCfg.Application.Component.Kafka.Topic,
		RequiredAcks: config.GatewayCfg.Application.Component.Kafka.RequiredAcks,
		MaxRetry:     config.GatewayCfg.Application.Component.Kafka.MaxRetries,
		GroupId:      config.GatewayCfg.Application.Component.Kafka.GroupID,
		DLQTopic:     config.GatewayCfg.Application.Component.Kafka.Dlq.TopicSuffix,
	}
	config.NewKafak(&conf)

	// 获取生产者连接
	producer := config.KafkaProducerTemplate()
	defer producer.Close()

	// 获取消费者连接
	consumer := config.KafkaConsumerTemplate()
	defer consumer.Close()

	// 获取死信队列连接
	dlq := config.KafkaDLQTemplate()
	defer dlq.Close()

	// 测试生产者
	err := service.NewProducer(producer, dlq).SendMessage(context.Background(), []byte("test"), "test")
	if err != nil {
		t.Errorf("send message failed: %v", err)
	}

	// 测试消费者
	err = service.NewReceviceMessage(consumer, dlq).StartReadMessage(t.Context(), "")
	if err != nil {
		t.Errorf("start read message failed: %v", err)
	}

}
