package main

import (
	"fmt"
	proto "github.com/chremoas/chremoas/proto"
	"github.com/chremoas/role-cmd/command"
	rolesrv "github.com/chremoas/role-srv/proto"
	permsrv "github.com/chremoas/perms-srv/proto"
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
		roleSrv:  config.LookupService("srv", "role"),
		permsSrv: config.LookupService("srv", "perms"),
		client:   service.Client()}

	proto.RegisterCommandHandler(service.Server(),
		command.NewCommand(name,
			&clientFactory,
		),
	)

	return nil
}

type clientFactory struct {
	roleSrv  string
	permsSrv string
	client   client.Client
}

func (c clientFactory) NewPermsClient() permsrv.PermissionsClient {
	return permsrv.NewPermissionsClient(c.permsSrv, c.client)
}

func (c clientFactory) NewRoleClient() rolesrv.RolesClient {
	return rolesrv.NewRolesClient(c.roleSrv, c.client)
}

func (c clientFactory) NewRuleClient() rolesrv.RulesClient {
	return rolesrv.NewRulesClient(c.roleSrv, c.client)
}

func (c clientFactory) NewFilterClient() rolesrv.FiltersClient {
	return rolesrv.NewFiltersClient(c.roleSrv, c.client)
}
