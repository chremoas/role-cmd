package main

import (
	"fmt"
	"github.com/abaeve/auth-admin-bot/command"
	uauthsvc "github.com/abaeve/auth-srv/proto"
	proto "github.com/abaeve/chremoas/proto"
	"github.com/abaeve/services-common/config"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
)

var Version string = "1.0.0"
var service micro.Service

func main() {
	service = config.NewService(Version, "template", initialize)

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}

// This function is a callback from the config.NewService function.  Read those docs
func initialize(config *config.Configuration) error {
	authSvcName := config.Bot.AuthSrvNamespace + "." + config.ServiceNames.AuthSrv
	clientFactory := clientFactory{name: authSvcName, client: service.Client()}

	proto.RegisterCommandHandler(service.Server(),
		command.NewCommand(config.Name,
			&clientFactory,
		),
	)

	return nil
}

type clientFactory struct {
	name   string
	client client.Client
}

func (c clientFactory) NewClient() uauthsvc.UserAuthenticationAdminClient {
	return uauthsvc.NewUserAuthenticationAdminClient(c.name, c.client)
}
