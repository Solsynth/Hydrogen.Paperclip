package grpc

import (
	idpb "git.solsynth.dev/hydrogen/passport/pkg/grpc/proto"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var Auth idpb.AuthClient

func ConnectPassport() error {
	addr := viper.GetString("passport.grpc_endpoint")
	if conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		return err
	} else {
		Auth = idpb.NewAuthClient(conn)
	}

	return nil
}
