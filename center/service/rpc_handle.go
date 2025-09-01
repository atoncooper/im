package service

import (
	"context"

	pb "github.com/atoncooper/im/proto"
	"github.com/segmentio/kafka-go"

	"github.com/redis/go-redis/v9"
)

type RPCHandle struct {
	pb.UnimplementedMessageServiceServer
	redis      *redis.ClusterClient
	kafkaWrite *kafka.Writer
}

func (r *RPCHandle) SendMessage(ctx context.Context, in *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	return nil, nil
}

// permission
//
// 检查是是否有权限发送消息
// 比如用户拉黑则无权限发送消息
// 比如用户被禁言则无权限发送消息
// 比如用户被踢出群聊则无权限发送消息
// 比如用户被禁言群聊则无权限发送消息
// 比如用户被踢出频道则无权限发送消息
//
// receiverId 接收者ID
// typ 类型 1:私聊 2:群聊 3:频道
func (r *RPCHandle) permission(ctx context.Context,
	senderId string,
	receiverId string, typ string) bool {

	// 构建闭包函数查询
	isBlackList := func(key string) bool {

		// TODO : 先查本地缓存

		vals, err := r.redis.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return false
		}
		for _, v := range vals {
			if v == senderId {
				return false
			}
		}
		return true
	}

	switch typ {
	case "group":
		// 群聊采用列表方式存储
		key := "groupBlackList:" + receiverId
		return isBlackList(key)
	case "channel":
		// TODO : 检查是否有权限发送消息
	default: // 默认私聊
		key := "blackList:" + receiverId
		return isBlackList(key)
	}

	return true
}

func (r *RPCHandle) pushStorage(ctx, typ string, message string) error { return nil }
