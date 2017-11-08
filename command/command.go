package command

import (
	"fmt"
	proto "github.com/abaeve/chremoas/proto"
	uauthsvc "github.com/abaeve/auth-srv/proto"
	"golang.org/x/net/context"
	"strings"
)

type ClientFactory interface {
	NewClient() uauthsvc.UserAuthenticationAdminClient
}

type Command struct {
	//Store anything you need the Help or Exec functions to have access to here
	name string
	factory ClientFactory
}

func (c *Command) Help(ctx context.Context, req *proto.HelpRequest, rsp *proto.HelpResponse) error {
	rsp.Usage = c.name
	rsp.Description = "Administrate Roles and shit"
	return nil
}

func (c *Command) Exec(ctx context.Context, req *proto.ExecRequest, rsp *proto.ExecResponse) error {
	var response string

	commandList := map[string]func([]string) string{
		"help":        help,
		"add_role":    addRole,
		"delete_role": notDefined,
	}

	f, ok := commandList[req.Args[1]]
	if ok {
		response = f(req.Args)
	} else {
		response = fmt.Sprintf("Not a valid subcommand: %s", req.Args[1])
	}

	rsp.Result = []byte(response)
	return nil
}

func help(args []string) string {
	return "This will be help info at some point.\nCan I do line breaks?"
}

func addRole(args []string) string {
	role := args[2]
	roleName := strings.Join(args[3:], " ")
	return fmt.Sprintf("adding: %s -- %s", role, roleName)
}

func notDefined(args []string) string {
	return "This command hasn't been defined yet"
}

func NewCommand(name string, factory ClientFactory) *Command {
	newCommand := Command{name: name, factory: factory}
	return &newCommand
}

//type UserAuthenticationAdminClient interface {
//	CharacterRoleAdd(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CharacterRoleRemove(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CorporationAllianceRoleAdd(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CorporationAllianceRoleRemove(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CorporationRoleAdd(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CorporationRoleRemove(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	AllianceRoleAdd(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	AllianceRoleRemove(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	AllianceCharacterLeadershipRoleAdd(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	AllianceCharacterLeadershipRoleRemove(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CorporationCharacterLeadershipRoleAdd(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//	CorporationCharacterLeadershipRoleRemove(ctx context.Context, in *AuthAdminRequest, opts ...client.CallOption) (*AuthAdminResponse, error)
//}