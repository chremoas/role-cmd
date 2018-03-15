package command

import (
	"bytes"
	"fmt"
	proto "github.com/chremoas/chremoas/proto"
	permsrv "github.com/chremoas/perms-srv/proto"
	rolesrv "github.com/chremoas/role-srv/proto"
	"golang.org/x/net/context"
	"strings"
)

type ClientFactory interface {
	NewPermsClient() permsrv.PermissionsClient
	NewRoleClient() rolesrv.RolesClient
	NewRuleClient() rolesrv.RulesClient
}

type command struct {
	funcptr func(ctx context.Context, request *proto.ExecRequest) string
	help    string
}

var cmdName = "role"
var commandList = map[string]command{
	"notDefined": {notDefined, ""},
	//"role_list":       {listRoles, "List all Roles"},
	//"role_add":        {addRole, "Add Role"},
	//"role_remove":     {removeRole, "Delete role"},
	"rule_list":   {listRules, "List all Rules"},
	"rule_add":    {addRule, "Add Rule"},
	"rule_remove": {removeRule, "Delete Rule"},
	"rule_info":   {ruleInfo, "Get Rule Info"},
	//"sync":            {syncRole, "Sync Roles to chat service"},
}

var clientFactory ClientFactory

type Command struct {
	//Store anything you need the Help or Exec functions to have access to here
	name    string
	factory ClientFactory
}

func (c *Command) Help(ctx context.Context, req *proto.HelpRequest, rsp *proto.HelpResponse) error {
	rsp.Usage = c.name
	rsp.Description = "Administrate Roles"
	return nil
}

func (c *Command) Exec(ctx context.Context, req *proto.ExecRequest, rsp *proto.ExecResponse) error {
	var response string

	if req.Args[1] == "help" {
		response = help(ctx, req)
	} else {
		f, ok := commandList[req.Args[1]]
		if ok {
			response = f.funcptr(ctx, req)
		} else {
			response = sendError(fmt.Sprintf("Not a valid subcommand: %s", req.Args[1]))
		}
	}

	rsp.Result = []byte(response)
	return nil
}

func help(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("Usage: !%s <subcommand> <arguments>\n", cmdName))
	buffer.WriteString("\nSubcommands:\n")

	for cmd := range commandList {
		if commandList[cmd].help != "" {
			buffer.WriteString(fmt.Sprintf("\t%s: %s\n", cmd, commandList[cmd].help))
		}
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func addRule(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 6 {
		return sendError("Usage: !role rule_add <rule_name> <role_name> <filterA> <filterB>")
	}

	name := req.Args[2]
	role := req.Args[3]
	filterA := req.Args[4]
	filterB := req.Args[5]

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	ruleClient := clientFactory.NewRuleClient()
	_, err = ruleClient.AddRule(ctx, &rolesrv.Rule{Name: name, Rule: &rolesrv.RuleInfo{Role: role, FilterA: filterA, FilterB: filterB}})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Added: %s\n", name))
}

func listRules(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	ruleClient := clientFactory.NewRuleClient()
	rules, err := ruleClient.GetRules(ctx, &rolesrv.NilMessage{})

	if err != nil {
		return sendFatal(err.Error())
	}

	buffer.WriteString("Rules:\n")
	for rule := range rules.Rules {
		buffer.WriteString(fmt.Sprintf("\t%s\n", rules.Rules[rule].Name))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func removeRule(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return sendError("Usage: !role rule_remove <rule_name>")
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	ruleClient := clientFactory.NewRuleClient()

	_, err = ruleClient.RemoveRule(ctx, &rolesrv.Rule{Name: req.Args[2]})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Removed: %s\n", req.Args[2]))
}

func ruleInfo(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return sendError("Usage: !role rule_info <rule_name>")
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	ruleClient := clientFactory.NewRuleClient()

	info, err := ruleClient.GetRule(ctx, &rolesrv.Rule{Name: req.Args[2]})
	if err != nil {
		return sendFatal(err.Error())
	}

	return fmt.Sprintf("```Name: %s\nRole: %s\nFilterA: %s\nFilterB: %s\n```",
		info.Name, info.Rule.Role, info.Rule.FilterA, info.Rule.FilterB)
}

//func syncRole(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//	var fromDiscord []string
//	var fromChremoas []string
//
//	rolesClient := clientFactory.NewRoleClient()
//	syncedRoles, err := rolesClient.SyncRoles(ctx, &rolesrv.SyncRolesRequest{})
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//		return fmt.Sprintf("```%s```", buffer.String())
//	}
//
//	if len(syncedRoles.Roles) == 0 {
//		return "```No roles to sync```"
//	}
//
//	for role := range syncedRoles.Roles {
//		if syncedRoles.Roles[role].Source == "Discord" {
//			fromDiscord = append(fromDiscord, syncedRoles.Roles[role].Name)
//		} else if syncedRoles.Roles[role].Source == "Chremoas" {
//			fromChremoas = append(fromChremoas, syncedRoles.Roles[role].Name)
//		} else {
//			// WTF
//			return "```WTF. Seriously.```"
//		}
//	}
//
//	if len(fromDiscord) != 0 {
//		buffer.WriteString("From Discord:\n")
//		for fd := range fromDiscord {
//			buffer.WriteString(fmt.Sprintf("\t%s\n", fromDiscord[fd]))
//		}
//	}
//
//	if len(fromChremoas) != 0 {
//		buffer.WriteString("From Chremoas:\n")
//		for fd := range fromChremoas {
//			buffer.WriteString(fmt.Sprintf("\t%s\n", fromChremoas[fd]))
//		}
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}
//
//func addRole(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//
//	rolesClient := clientFactory.NewRoleClient()
//	roleName := req.Args[2]
//	chatServiceGroup := strings.Join(req.Args[3:], " ")
//
//	if len(chatServiceGroup) > 0 && chatServiceGroup[0] == '"' {
//		chatServiceGroup = chatServiceGroup[1:]
//	}
//	if len(chatServiceGroup) > 0 && chatServiceGroup[len(chatServiceGroup)-1] == '"' {
//		chatServiceGroup = chatServiceGroup[:len(chatServiceGroup)-1]
//	}
//
//	_, err := rolesClient.AddRole(ctx, &rolesrv.AddRoleRequest{
//		Role: &rolesrv.DiscordRole{
//			Name:     roleName,
//			RoleNick: chatServiceGroup,
//		},
//	})
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//	} else {
//		buffer.WriteString(fmt.Sprintf("Adding role '%s'\n", chatServiceGroup))
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}
//
//func deleteRole(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//	rolesClient := clientFactory.NewRoleClient()
//	roleName := req.Args[2]
//
//	_, err := rolesClient.RemoveRole(ctx, &rolesrv.RemoveRoleRequest{
//		Name: roleName,
//	})
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//	} else {
//		buffer.WriteString(fmt.Sprintf("Deleting role: %s\n", roleName))
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}
//
//func listRoles(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//	rolesClient := clientFactory.NewRoleClient()
//
//	output, err := rolesClient.GetRoles(ctx, &rolesrv.GetRolesRequest{})
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//	} else {
//		if output.String() == "" {
//			buffer.WriteString("There are no roles defined")
//		} else {
//			for role := range output.Roles {
//				buffer.WriteString(fmt.Sprintf("%s: %s\n",
//					output.Roles[role].RoleNick,
//					output.Roles[role].Name,
//				))
//			}
//		}
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}

//func myID(ctx context.Context, req *proto.ExecRequest) string {
//	senderID := strings.Split(req.Sender, ":")[1]
//
//	authClient := clientFactory.NewClient()
//	response, err := authClient.GetRoles(ctx, &uauthsvc.GetRolesRequest{UserId: senderID})
//	if err != nil {
//		return err.Error()
//	}
//
//	fmt.Printf("%+v\n", response.Roles)
//
//	if len(response.GetRoles()) == 0 {
//		return fmt.Sprintf("<@%s> You have no roles", senderID)
//	}
//	return strings.Join(response.GetRoles(), " ")
//}

func notDefined(ctx context.Context, req *proto.ExecRequest) string {
	return "This command hasn't been defined yet"
}

func NewCommand(name string, factory ClientFactory) *Command {
	clientFactory = factory
	newCommand := Command{name: name, factory: factory}
	return &newCommand
}

//func addDRole(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//	name := strings.Join(req.Args[2:], " ")
//
//	client := clientFactory.NewDiscordGatewayClient()
//	output, err := client.CreateRole(ctx, &discord.CreateRoleRequest{Name: name})
//	//string GuildId = 1;
//	//string Name = 2;
//	//int32 Color = 3;
//	//bool Hoist = 4;
//	//int32 Permissions = 5;
//	//bool Mentionable = 6;
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//	} else {
//		buffer.WriteString(output.String())
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}
//
//func listDRoles(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//
//	client := clientFactory.NewDiscordGatewayClient()
//	output, err := client.GetAllRoles(ctx, &discord.GuildObjectRequest{})
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//	} else {
//
//		w := tabwriter.NewWriter(&buffer, 0, 0, 1, ' ', tabwriter.Debug)
//		fmt.Fprintln(w, "Position\tName")
//		for _, v := range output.Roles {
//			foo := fmt.Sprintf("%d\t%s", v.Position, v.Name)
//			fmt.Fprintln(w, foo)
//		}
//		w.Flush()
//
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}
//
//func deleteDRole(ctx context.Context, req *proto.ExecRequest) string {
//	var buffer bytes.Buffer
//	name := strings.Join(req.Args[2:], " ")
//
//	client := clientFactory.NewDiscordGatewayClient()
//	output, err := client.DeleteRole(ctx, &discord.DeleteRoleRequest{Name: name})
//	//string GuildId = 1;
//	//string Name = 2;
//	//int32 Color = 3;
//	//bool Hoist = 4;
//	//int32 Permissions = 5;
//	//bool Mentionable = 6;
//
//	if err != nil {
//		buffer.WriteString(err.Error())
//	} else {
//		buffer.WriteString(output.String())
//	}
//
//	return fmt.Sprintf("```%s```", buffer.String())
//}

func sendSuccess(message string) string {
	return fmt.Sprintf(":white_check_mark: %s", message)
}

func sendError(message string) string {
	return fmt.Sprintf(":warning: %s", message)
}

func sendFatal(message string) string {
	return fmt.Sprintf(":octagonal_sign: %s", message)
}

func canPerform(ctx context.Context, req *proto.ExecRequest, perms []string) (bool, error) {
	permsClient := clientFactory.NewPermsClient()

	sender := strings.Split(req.Sender, ":")
	canPerform, err := permsClient.Perform(ctx,
		&permsrv.PermissionsRequest{User: sender[1], PermissionsList: perms})

	if err != nil {
		return false, err
	}
	return canPerform.CanPerform, nil
}
