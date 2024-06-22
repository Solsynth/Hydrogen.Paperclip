package grpc

import (
	"git.solsynth.dev/hydrogen/paperclip/pkg/proto"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"net"
)

type Server struct {
	proto.UnimplementedAttachmentsServer
}

var S *grpc.Server

func NewGRPC() {
	S = grpc.NewServer()

	proto.RegisterAttachmentsServer(S, &Server{})
	health.RegisterHealthServer(S, &Server{})

	reflection.Register(S)
}

func ListenGRPC() error {
	listener, err := net.Listen("tcp", viper.GetString("grpc_bind"))
	if err != nil {
		return err
	}

	return S.Serve(listener)
}
