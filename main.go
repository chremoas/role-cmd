package main

import (
	"fmt"
	proto "github.com/chremoas/chremoas/proto"
	permsrv "github.com/chremoas/perms-srv/proto"
	"github.com/chremoas/role-cmd/command"
	rolesrv "github.com/chremoas/role-srv/proto"
	"github.com/chremoas/services-common/config"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"go.uber.org/zap"
)

var Version = "SET ME YOU KNOB"
var service micro.Service
var logger *zap.Logger
var name = "role"

func main() {
	service = config.NewService(Version, "cmd", name, initialize)

	// TODO pick stuff up from the config
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	logger.Info("Initialized logger")

	if err := service.Run(); err != nil {
		fmt.Println(err)
	}
}

// This function is a callback from the config.NewService function.  Read those docs
func initialize(config *config.Configuration) error {
	clientFactory := clientFactory{
		roleSrv:  config.LookupService("srv", "role"),
		permsSrv: config.LookupService("srv", "perms"),
		client:   service.Client(),
		Logger:   logger,
	}

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

func (c clientFactory) NewPermsClient() permsrv.PermissionsService {
	return permsrv.NewPermissionsService(c.permsSrv, c.client)
}

func (c clientFactory) NewRoleClient() rolesrv.RolesService {
	return rolesrv.NewRolesService(c.roleSrv, c.client)
}

func (c clientFactory) NewLogger() *zap.Logger {
	return logger
}
