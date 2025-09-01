package service

import (
	"context"
	"encoding/json"
	"errors"
	"gateway/dto"
	"gateway/utils"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

type receviceMessage struct {
	reader *kafka.Reader
	dlq    *kafka.Writer
	wg     sync.WaitGroup
}

func NewReceviceMessage(reader *kafka.Reader, write *kafka.Writer) *receviceMessage {
	return &receviceMessage{reader: reader, dlq: write}
}

const MAX_RETRY = 3

func (r *receviceMessage) StartReadMessage(ctx context.Context, nodeId string) error {
	if r.reader == nil {
		return errors.New("kafka reader is nil")
	}

	r.wg.Add(1)
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Default().Println("[INFO] 消息服务接受服务正在被关闭...")
			return ctx.Err()
		default:
			readCtx, cancel := context.WithTimeout(ctx, 35*time.Second)
			msg, err := r.reader.ReadMessage(readCtx)
			cancel() // 立即取消上下文，避免泄漏

			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// 改进错误处理，区分不同类型的错误
				if errors.Is(err, context.DeadlineExceeded) {
					// 超时错误通常是正常的，不需要高频率日志
					log.Default().Println("[DEBUG] 消息读取超时，继续等待新消息")
				} else {
					log.Default().Printf("[ERROR] 读取消息失败: %v", err)
				}
				// 短暂休眠避免CPU空转
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// 处理消息
			var message *dto.MessageDTO
			if err = json.Unmarshal(msg.Value, &message); err != nil {
				// TODO : 死信队列或者丢回消息队列
				err = r.dlq.WriteMessages(ctx, msg)
				if err != nil {
					// TODO : 消息可能丢失注意
				}
			}

			// 发送至正在连接的conn
			meta, err := utils.StatusTemplate().GetStatus(message.ReceiverId)
			if err != nil {
				// 扔进死信队列
				err = r.dlq.WriteMessages(ctx, msg)
				if err != nil {
					// TODO : 消息可能丢失注意
				}
			}
			conn, ok := utils.PoolsOpsTemplate().GetConnTemplate(meta.UserRemoteAddr)
			if !ok {
				err = r.dlq.WriteMessages(ctx, msg)
				if err != nil {
					// TODO : 消息可能丢失注意
				}
			}
			conn.WriteMessage(websocket.TextMessage, msg.Value)
			r.reader.CommitMessages(ctx, msg) // 手动ACK
		}
	}

}

func (r *receviceMessage) Wait() {
	r.wg.Wait()
}
