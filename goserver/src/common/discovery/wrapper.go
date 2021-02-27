package discovery

import (
	"common/proto/config"
	"google.golang.org/grpc"
)

const (
	ConfigServer = "config"
)

func RegisterConfigServer(env string, implementation config.ConfigServer) error {
	listener, addr, err := getListener()
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	config.RegisterConfigServer(srv, implementation)
	go srv.Serve(listener)
	Register(env, &Service{Name: ConfigServer, Addr: addr})
	return nil
}

func ResolverConfigServer(env string) config.ConfigClient {
	conn, _ := Resolver(env, ConfigServer, DependNormal)
	return config.NewConfigClient(conn)
}
