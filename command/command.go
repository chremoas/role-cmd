package command

import (
	"bytes"
	"fmt"
	proto "github.com/chremoas/chremoas/proto"
	permsrv "github.com/chremoas/perms-srv/proto"
	rclient "github.com/chremoas/role-srv/client"
	rolesrv "github.com/chremoas/role-srv/proto"
	"github.com/chremoas/services-common/args"
	common "github.com/chremoas/services-common/command"
	"golang.org/x/net/context"
	"strings"
)

type ClientFactory interface {
	NewPermsClient() permsrv.PermissionsService
	NewRoleClient() rolesrv.RolesService
}

var role rclient.Roles
var cmdName = "role"
var clientFactory ClientFactory

type Command struct {
	//Store anything you need the Help or Exec functions to have access to here
	name    string
	factory ClientFactory
}

func (c *Command) Help(ctx context.Context, req *proto.HelpRequest, rsp *proto.HelpResponse) error {
	rsp.Usage = c.name
	rsp.Description = "Administrate Roles, Rules and Filters"
	return nil
}

func (c *Command) Exec(ctx context.Context, req *proto.ExecRequest, rsp *proto.ExecResponse) error {
	cmd := args.NewArg(cmdName)
	cmd.Add("list", &args.Command{listRoles, "List all Roles"})
	cmd.Add("create", &args.Command{addRole, "Add Role"})
	cmd.Add("destroy", &args.Command{removeRole, "Delete role"})
	cmd.Add("info", &args.Command{roleInfo, "Get Role Info"})
	cmd.Add("keys", &args.Command{roleKeys, "Get valid role keys"})
	cmd.Add("types", &args.Command{roleTypes, "Get valid role types"})
	cmd.Add("sync", &args.Command{syncRoles, "Sync Roles to chat service"})
	err := cmd.Exec(ctx, req, rsp)

	// I don't 100% love this, but it'll do for now. -brian
	if err != nil {
		rsp.Result = []byte(common.SendError(err.Error()))
	}
	return nil
}

func roleKeys(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	roleClient := clientFactory.NewRoleClient()
	keys, err := roleClient.GetRoleKeys(ctx, &rolesrv.NilMessage{})
	if err != nil {
		return common.SendFatal(err.Error())
	}

	buffer.WriteString("Keys:\n")
	for key := range keys.Value {
		buffer.WriteString(fmt.Sprintf("\t%s\n", keys.Value[key]))
	}

	return common.SendSuccess(fmt.Sprintf("```%s```\n", buffer.String()))
}

func roleTypes(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	roleClient := clientFactory.NewRoleClient()
	keys, err := roleClient.GetRoleTypes(ctx, &rolesrv.NilMessage{})
	if err != nil {
		return common.SendFatal(err.Error())
	}

	buffer.WriteString("Types:\n")
	for key := range keys.Value {
		buffer.WriteString(fmt.Sprintf("\t%s\n", keys.Value[key]))
	}

	return common.SendSuccess(fmt.Sprintf("```%s```\n", buffer.String()))
}

func addRole(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 5 {
		return common.SendError("Usage: !role create <role_name> <filter> <role_description>")
	}

	roleName := strings.Join(req.Args[4:], " ")

	if common.IsDiscordUser(req.Args[2]) {
		return common.SendError("Discord users may not be roles")
	}

	if common.IsDiscordUser(roleName) {
		return common.SendError("Discord users may not be descriptions")
	}


	return role.AddRole(ctx,
		req.Sender,
		req.Args[2], // shortName
		"discord",   // roleType
		req.Args[3], // filterA
		"wildcard",  // filterB
		false,       // Is this SIG joinable? (Not a SIG, so no)
		roleName, // roleName
		false, // Is this a SIG?
	)
}

func listRoles(ctx context.Context, req *proto.ExecRequest) string {
	var all = false
	if len(req.Args) == 3 {
		if req.Args[2] == "all" {
			all = true
		}
	}

	return role.ListRoles(ctx, all, false)
}

func removeRole(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return common.SendError("Usage: !role destroy <role_name>")
	}

	return role.RemoveRole(ctx, req.Sender, req.Args[2], false)
}

func roleInfo(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return common.SendError("Usage: !role info <role_name>")
	}

	return role.RoleInfo(ctx, req.Sender, req.Args[2], false)
}

func syncRoles(ctx context.Context, req *proto.ExecRequest) string {
	return role.SyncRoles(ctx)
}

func NewCommand(name string, factory ClientFactory) *Command {
	clientFactory = factory
	role = rclient.Roles{
		RoleClient:  clientFactory.NewRoleClient(),
		PermsClient: clientFactory.NewPermsClient(),
		Permissions: common.Permissions{Client: clientFactory.NewPermsClient(), PermissionsList: []string{"role_admins"}},
	}

	return &Command{name: name, factory: factory}
}
