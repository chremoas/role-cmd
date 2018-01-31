package main

import (
	"fmt"
	rolesvc "github.com/chremoas/role-srv/proto"
	proto "github.com/chremoas/chremoas/proto"
	"github.com/chremoas/role-cmd/command"
	"github.com/chremoas/services-common/config"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
)

var Version = "1.0.0"
var service micro.Service
var name = "role"

func main() {
	service = config.NewService(Version, "cmd", name, initialize)

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}

// This function is a callback from the config.NewService function.  Read those docs
func initialize(config *config.Configuration) error {
	clientFactory := clientFactory{
		roleSrv:        config.LookupService("srv", "role"),
		client:         service.Client()}

	proto.RegisterCommandHandler(service.Server(),
		command.NewCommand(name,
			&clientFactory,
		),
	)

	return nil
}

type clientFactory struct {
	roleSrv        string
	client         client.Client
}

func (c clientFactory) NewRoleClient() rolesvc.RolesClient {
	return rolesvc.NewRolesClient(c.roleSrv, c.client)
}
