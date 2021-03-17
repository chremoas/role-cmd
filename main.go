package main

import (
	"fmt"

	proto "github.com/chremoas/chremoas/proto"
	permsrv "github.com/chremoas/perms-srv/proto"
	rolesrv "github.com/chremoas/role-srv/proto"
	"github.com/chremoas/services-common/config"
	chremoasPrometheus "github.com/chremoas/services-common/prometheus"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"go.uber.org/zap"

	"github.com/chremoas/role-cmd/command"
)

var (
	Version = "SET ME YOU KNOB"
	service micro.Service
	logger  *zap.Logger
	name    = "role"
)

func main() {
	var err error

	// TODO pick stuff up from the config
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	logger.Info("Initialized logger")

	go chremoasPrometheus.PrometheusExporter(logger)

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
		client:   service.Client(),
	}

	proto.RegisterCommandHandler(service.Server(),
		command.NewCommand(name,
			&clientFactory,
			logger,
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
