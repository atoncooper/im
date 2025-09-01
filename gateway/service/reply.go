package service

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type producer struct {
	writer *kafka.Writer
	dlq    *kafka.Writer
}

func NewProducer(writer *kafka.Writer, dq *kafka.Writer) *producer {
	return &producer{
		writer: writer,
		dlq:    dq,
	}
}

const MAX_SEND_RETRY = 3

// SendMessage
//
// 转发消息到对应的gateway节点处理消息
func (p *producer) SendMessage(
	ctx context.Context,
	msg any,
	nodeId string,
) error {

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(nodeId),
		Value: msg.([]byte),
	})
	if err != nil {

		for i := 0; i < MAX_SEND_RETRY; i++ {
			select {
			case <-time.After(time.Duration(i) * 100 * time.Microsecond):
				// TODO
			case <-ctx.Done():
				return ctx.Err()
			}
			err = p.writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(nodeId),
				Value: msg.([]byte),
			})
			if err == nil {
				break
			}
		}
		// 写入死信队列
		err = p.dlq.WriteMessages(ctx, kafka.Message{
			Key:   []byte(nodeId),
			Value: msg.([]byte),
		})
		if err != nil {
			return err
		}
	}

	return nil
}
