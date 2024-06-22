package gap

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Register() error {
	cfg := api.DefaultConfig()
	cfg.Address = viper.GetString("consul.addr")

	client, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	httpBind := strings.SplitN(viper.GetString("bind"), ":", 2)
	grpcBind := strings.SplitN(viper.GetString("grpc_bind"), ":", 2)

	outboundIp, _ := GetOutboundIP()
	port, _ := strconv.Atoi(httpBind[1])

	registration := new(api.AgentServiceRegistration)
	registration.ID = viper.GetString("id")
	registration.Name = "Hydrogen.Paperclip"
	registration.Address = outboundIp.String()
	registration.Port = port
	registration.Check = &api.AgentServiceCheck{
		GRPC:                           fmt.Sprintf("%s:%s", outboundIp, grpcBind[1]),
		Timeout:                        "5s",
		Interval:                       "1m",
		DeregisterCriticalServiceAfter: "3m",
	}

	return client.Agent().ServiceRegister(registration)
}

func DiscoverPassport() (*grpc.ClientConn, error) {
	target := fmt.Sprintf("consul://%s/Hydrogen.Passport", viper.GetString("consul.addr"))
	return grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
	)
}
