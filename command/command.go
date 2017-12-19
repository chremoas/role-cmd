package command

import (
	"bytes"
	"fmt"
	uauthsvc "github.com/chremoas/auth-srv/proto"
	discord "github.com/chremoas/discord-gateway/proto"
	proto "github.com/chremoas/chremoas/proto"
	"golang.org/x/net/context"
	"strings"
)

type ClientFactory interface {
	NewClient() uauthsvc.UserAuthenticationClient
	NewAdminClient() uauthsvc.UserAuthenticationAdminClient
	NewEntityQueryClient() uauthsvc.EntityQueryClient
	NewEntityAdminClient() uauthsvc.EntityAdminClient
	NewDiscordGatewayClient() discord.DiscordGatewayClient
}

var clientFactory ClientFactory

type Command struct {
	//Store anything you need the Help or Exec functions to have access to here
	name    string
	factory ClientFactory
}

func (c *Command) Help(ctx context.Context, req *proto.HelpRequest, rsp *proto.HelpResponse) error {
	rsp.Usage = c.name
	rsp.Description = "Administrate Roles and shit"
	return nil
}

func (c *Command) Exec(ctx context.Context, req *proto.ExecRequest, rsp *proto.ExecResponse) error {
	var response string

	commandList := map[string]func(context.Context, *proto.ExecRequest) string{
		"help":        help,
		"list":  listRoles,
		"dlist":  listDRoles,
		"add":    addRole,
		"delete": deleteRole,
		"my_id":       myID,
		"notDefined":  notDefined,
	}

	f, ok := commandList[req.Args[1]]
	if ok {
		response = f(ctx, req)
	} else {
		response = fmt.Sprintf("Not a valid subcommand: %s", req.Args[1])
	}

	rsp.Result = []byte(response)
	return nil
}

func help(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	buffer.WriteString("Usage: !admin <subcommand> <arguments>\n")
	buffer.WriteString("\nSubcommands:\n")
	buffer.WriteString("\tlist: Lists all roles\n")
	buffer.WriteString("\tadd <role name> <chat service name>: Add Role\n")
	buffer.WriteString("\tdelete <role name>: Delete Role\n")
	buffer.WriteString("\tdlist: Get roles list from Discord, not Chremoas\n")
	buffer.WriteString("\thelp: This text\n")

	return fmt.Sprintf("```%s```", buffer.String())
}

func addRole(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	client := clientFactory.NewEntityAdminClient()
	roleName := req.Args[2]
	chatServiceGroup := strings.Join(req.Args[3:], " ")

	if len(chatServiceGroup) > 0 && chatServiceGroup[0] == '"' {
		chatServiceGroup = chatServiceGroup[1:]
	}
	if len(chatServiceGroup) > 0 && chatServiceGroup[len(chatServiceGroup)-1] == '"' {
		chatServiceGroup = chatServiceGroup[:len(chatServiceGroup)-1]
	}

	output, err := client.RoleUpdate(ctx, &uauthsvc.RoleAdminRequest{
		Role:      &uauthsvc.Role{RoleName: roleName, ChatServiceGroup: chatServiceGroup},
		Operation: uauthsvc.EntityOperation_ADD_OR_UPDATE,
	})

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		buffer.WriteString(output.String())
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func deleteRole(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	client := clientFactory.NewEntityAdminClient()
	roleName := req.Args[2]

	output, err := client.RoleUpdate(ctx, &uauthsvc.RoleAdminRequest{
		Role:      &uauthsvc.Role{RoleName: roleName, ChatServiceGroup: "Doesn't matter"},
		Operation: uauthsvc.EntityOperation_REMOVE,
	})

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		buffer.WriteString(output.String())
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func listDRoles(ctx context.Context, req *proto.ExecRequest) string {
	client := clientFactory.NewDiscordGatewayClient()
	output, err := client.GetAllRoles(ctx, &discord.GuildObjectRequest{GuildId: "374983726763081738"})
	var buffer bytes.Buffer

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		fmt.Sprintf("output: %+v\n", output)
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func listRoles(ctx context.Context, req *proto.ExecRequest) string {
	client := clientFactory.NewEntityQueryClient()
	output, err := client.GetRoles(ctx, &uauthsvc.EntityQueryRequest{})
	var buffer bytes.Buffer

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		if output.String() == "" {
			buffer.WriteString("There are no roles defined")
		} else {
			for role := range output.List {
				buffer.WriteString(fmt.Sprintf("%s: %s\n",
					output.List[role].RoleName,
					output.List[role].ChatServiceGroup,
				))
			}
		}
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func myID(ctx context.Context, req *proto.ExecRequest) string {
	senderID := strings.Split(req.Sender, ":")[1]

	authClient := clientFactory.NewClient()
	response, err := authClient.GetRoles(ctx, &uauthsvc.GetRolesRequest{UserId: senderID})
	if err != nil {
		return err.Error()
	}

	fmt.Printf("%+v\n", response.Roles)

	if len(response.GetRoles()) == 0 {
		return fmt.Sprintf("<@%s> You have no roles", senderID)
	}
	return strings.Join(response.GetRoles(), " ")
}

func notDefined(ctx context.Context, req *proto.ExecRequest) string {
	return "This command hasn't been defined yet"
}

func NewCommand(name string, factory ClientFactory) *Command {
	clientFactory = factory
	newCommand := Command{name: name, factory: factory}
	return &newCommand
}
