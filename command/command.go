package command

import (
	"bytes"
	"fmt"
	uauthsvc "github.com/chremoas/auth-srv/proto"
	proto "github.com/chremoas/chremoas/proto"
	discord "github.com/chremoas/discord-gateway/proto"
	"golang.org/x/net/context"
	"regexp"
	"strings"
	"text/tabwriter"
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
		"help":   help,
		"list":   listRoles,
		"add":    addRole,
		"delete": deleteRole,
		"sync":   syncRole,
		// going to move the discord ones later
		"dlist":      listDRoles,
		"dadd":       addDRole,
		"ddelete":    deleteDRole,
		"my_id":      myID,
		"notDefined": notDefined,
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

	buffer.WriteString("Usage: !role <subcommand> <arguments>\n")
	buffer.WriteString("\nSubcommands:\n")
	buffer.WriteString("\tlist: Lists all roles\n")
	buffer.WriteString("\tadd <role name> <chat service name>: Add Role\n")
	buffer.WriteString("\tdelete <role name>: Delete Role\n")
	buffer.WriteString("\tdebug: Debug commands\n")
	buffer.WriteString("\thelp: This text\n")

	return fmt.Sprintf("```%s```", buffer.String())
}

func syncRole(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	var matchSpace = regexp.MustCompile(`\s`)
	var matchDBError = regexp.MustCompile(`^Error 1062:`)
	var matchDiscordError = regexp.MustCompile(`^The role '.*' already exists$`)

	//listDRoles(ctx, req)
	discordClient := clientFactory.NewDiscordGatewayClient()
	discordRoles, err := discordClient.GetAllRoles(ctx, &discord.GuildObjectRequest{})

	if err != nil {
		buffer.WriteString(err.Error())
		return fmt.Sprintf("```%s```", buffer.String())
	}

	//listRoles(ctx, req)
	chremoasClient := clientFactory.NewEntityQueryClient()
	chremoasRoles, err := chremoasClient.GetRoles(ctx, &uauthsvc.EntityQueryRequest{})

	if err != nil {
		buffer.WriteString(err.Error())
		return fmt.Sprintf("```%s```", buffer.String())
	}

	for dr := range discordRoles.Roles {
		chremoasClient := clientFactory.NewEntityAdminClient()

		_, err := chremoasClient.RoleUpdate(ctx, &uauthsvc.RoleAdminRequest{
			Role:      &uauthsvc.Role{ChatServiceGroup: discordRoles.Roles[dr].Name, RoleName: matchSpace.ReplaceAllString(discordRoles.Roles[dr].Name, "_")},
			Operation: uauthsvc.EntityOperation_ADD_OR_UPDATE,
		})

		if err != nil {
			if !matchDBError.MatchString(err.Error()) {
				buffer.WriteString(err.Error() + "\n")
			}
		} else {
			buffer.WriteString(fmt.Sprintf("Syncing role '%s' from Discord to Chremoas\n", discordRoles.Roles[dr].Name))
		}
	}

	for cr := range chremoasRoles.List {
		discordClient := clientFactory.NewDiscordGatewayClient()
		_, err := discordClient.CreateRole(ctx, &discord.CreateRoleRequest{Name: chremoasRoles.List[cr].ChatServiceGroup})

		if err != nil {
			if !matchDiscordError.MatchString(err.Error()) {
				buffer.WriteString(err.Error() + "\n")
			}
		} else {
			buffer.WriteString(fmt.Sprintf("Syncing role '%s' from Chremoas to Discord\n", chremoasRoles.List[cr].ChatServiceGroup))
		}
	}

	if buffer.Len() == 0 {
		buffer.WriteString("No roles needed to be synced")
	}

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
	var dRoleName string
	chremoasClient := clientFactory.NewEntityAdminClient()
	discordClient := clientFactory.NewDiscordGatewayClient()
	roleName := req.Args[2]

	chremoasQueryClient := clientFactory.NewEntityQueryClient()
	chremoasRoles, err := chremoasQueryClient.GetRoles(ctx, &uauthsvc.EntityQueryRequest{})

	for cr := range chremoasRoles.List {
		if chremoasRoles.List[cr].RoleName == roleName {
			dRoleName = chremoasRoles.List[cr].ChatServiceGroup
		}
	}

	_, err = chremoasClient.RoleUpdate(ctx, &uauthsvc.RoleAdminRequest{
		Role:      &uauthsvc.Role{RoleName: roleName, ChatServiceGroup: "Doesn't matter"},
		Operation: uauthsvc.EntityOperation_REMOVE,
	})

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		buffer.WriteString(fmt.Sprintf("Deleting role from Chremoas: %s\n", roleName))
	}

	_, err = discordClient.DeleteRole(ctx, &discord.DeleteRoleRequest{Name: dRoleName})

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		buffer.WriteString(fmt.Sprintf("Deleting role from Discord: %s\n", dRoleName))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func addDRole(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	name := strings.Join(req.Args[2:], " ")

	client := clientFactory.NewDiscordGatewayClient()
	output, err := client.CreateRole(ctx, &discord.CreateRoleRequest{Name: name})
	//string GuildId = 1;
	//string Name = 2;
	//int32 Color = 3;
	//bool Hoist = 4;
	//int32 Permissions = 5;
	//bool Mentionable = 6;

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		buffer.WriteString(output.String())
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func listDRoles(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	client := clientFactory.NewDiscordGatewayClient()
	output, err := client.GetAllRoles(ctx, &discord.GuildObjectRequest{})

	if err != nil {
		buffer.WriteString(err.Error())
	} else {

		w := tabwriter.NewWriter(&buffer, 0, 0, 1, ' ', tabwriter.Debug)
		fmt.Fprintln(w, "Position\tName")
		for _, v := range output.Roles {
			foo := fmt.Sprintf("%d\t%s", v.Position, v.Name)
			fmt.Fprintln(w, foo)
		}
		w.Flush()

	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func deleteDRole(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	name := strings.Join(req.Args[2:], " ")

	client := clientFactory.NewDiscordGatewayClient()
	output, err := client.DeleteRole(ctx, &discord.DeleteRoleRequest{Name: name})
	//string GuildId = 1;
	//string Name = 2;
	//int32 Color = 3;
	//bool Hoist = 4;
	//int32 Permissions = 5;
	//bool Mentionable = 6;

	if err != nil {
		buffer.WriteString(err.Error())
	} else {
		buffer.WriteString(output.String())
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
