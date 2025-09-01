package services

import (
	"context"
	"errors"
	"strconv"

	pb "github.com/atoncooper/im/proto/seq"
	"github.com/redis/go-redis/v9"
)

type ServerHandle struct {
	pb.UnimplementedSequenceServiceServer
	Redis *redis.ClusterClient
}

func (s *ServerHandle) GenerateMessageId(ctx context.Context, in *pb.Empty) (*pb.MessageIdResponse, error) {
	msgId := GeneratorFactory().GenerateMessageID()
	if msgId == 0 {
		return nil, errors.New("generate message id failed")
	}

	resp := &pb.MessageIdResponse{Id: strconv.FormatInt(msgId, 10)}

	return resp, nil
}

func (s *ServerHandle) GenerateMessageSeq(ctx context.Context, in *pb.MessageSeqRequest) (*pb.MessageResponse, error) {
	msgId, err := GeneratorFactory().GenerateSeq(in.SenderId, in.ReceiverId)
	if err != nil {
		return nil, err
	}

	resp := &pb.MessageResponse{Seq: msgId}
	return resp, nil
}
