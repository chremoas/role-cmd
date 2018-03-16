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
	NewFilterClient() rolesrv.FiltersClient
}

type command struct {
	funcptr func(ctx context.Context, request *proto.ExecRequest) string
	help    string
}

var cmdName = "role"
var commandList = map[string]command{
	// Roles
	"role_list":   {listRoles, "List all Roles"},
	"role_add":    {addRole, "Add Role"},
	"role_remove": {removeRole, "Delete role"},
	"role_info":   {roleInfo, "Get Role Info"},
	"role_keys":   {roleKeys, "Get valid role keys"},
	"role_types":  {roleTypes, "Get valid role types"},
	//"sync":            {syncRole, "Sync Roles to chat service"},

	// Rules
	"rule_list":   {listRules, "List all Rules"},
	"rule_add":    {addRule, "Add Rule"},
	"rule_remove": {removeRule, "Delete Rule"},
	"rule_info":   {ruleInfo, "Get Rule Info"},

	// Filters
	"filter_list":   {listFilters, "List all Filters"},
	"filter_add":    {addFilter, "Add Filter"},
	"filter_remove": {removeFilter, "Delete Filter"},
	"member_list":   {listMembers, "List all Filter Members"},
	"member_add":    {addMember, "Add Filter Member"},
	"member_remove": {removeMember, "Remove Filter Member"},
	//"filter_info":   {filterInfo, "Get Rule Info"},
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

func roleKeys(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	roleClient := clientFactory.NewRoleClient()
	keys, err := roleClient.GetRoleKeys(ctx, &rolesrv.NilMessage{})
	if err != nil {
		return sendFatal(err.Error())
	}

	buffer.WriteString("Keys:\n")
	for key := range keys.Value {
		buffer.WriteString(fmt.Sprintf("\t%s\n", keys.Value[key]))
	}

	return sendSuccess(fmt.Sprintf("```%s```\n", buffer.String()))
}

func roleTypes(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer

	roleClient := clientFactory.NewRoleClient()
	keys, err := roleClient.GetRoleTypes(ctx, &rolesrv.NilMessage{})
	if err != nil {
		return sendFatal(err.Error())
	}

	buffer.WriteString("Types:\n")
	for key := range keys.Value {
		buffer.WriteString(fmt.Sprintf("\t%s\n", keys.Value[key]))
	}

	return sendSuccess(fmt.Sprintf("```%s```\n", buffer.String()))
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

func addRole(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 5 {
		return sendError("Usage: !role role_add <role_short_name> <role_type> <role_name>")
	}

	roleShortName := req.Args[2]
	roleType := req.Args[3]
	roleName := strings.Join(req.Args[4:], " ")

	if len(roleName) > 0 && roleName[0] == '"' {
		roleName = roleName[1:]
	}

	if len(roleName) > 0 && roleName[len(roleName)-1] == '"' {
		roleName = roleName[:len(roleName)-1]
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	roleClient := clientFactory.NewRoleClient()
	_, err = roleClient.AddRole(ctx, &rolesrv.Role{ShortName: roleShortName, Type: roleType, Name: roleName})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Added: %s\n", roleShortName))
}

func addFilter(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 4 {
		return sendError("Usage: !role filter_add <filter_name> <filter_description>")
	}

	filterName := req.Args[2]
	filterDescription := strings.Join(req.Args[3:], " ")

	if len(filterDescription) > 0 && filterDescription[0] == '"' {
		filterDescription = filterDescription[1:]
	}

	if len(filterDescription) > 0 && filterDescription[len(filterDescription)-1] == '"' {
		filterDescription = filterDescription[:len(filterDescription)-1]
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	filterClient := clientFactory.NewFilterClient()
	_, err = filterClient.AddFilter(ctx, &rolesrv.Filter{Name: filterName, Description: filterDescription})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Added: %s\n", filterName))
}

func listRules(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	ruleClient := clientFactory.NewRuleClient()
	rules, err := ruleClient.GetRules(ctx, &rolesrv.NilMessage{})

	if err != nil {
		return sendFatal(err.Error())
	}

	if len(rules.Rules) == 0 {
		return sendError("No Rules\n")
	}

	buffer.WriteString("Rules:\n")
	for rule := range rules.Rules {
		buffer.WriteString(fmt.Sprintf("\t%s\n", rules.Rules[rule].Name))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func listRoles(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	roleClient := clientFactory.NewRoleClient()
	roles, err := roleClient.GetRoles(ctx, &rolesrv.NilMessage{})

	if err != nil {
		return sendFatal(err.Error())
	}

	if len(roles.Roles) == 0 {
		return sendError("No Roles\n")
	}

	buffer.WriteString("Roles:\n")
	for role := range roles.Roles {
		buffer.WriteString(fmt.Sprintf("\t%s\n", roles.Roles[role].Name))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func listFilters(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	filterClient := clientFactory.NewFilterClient()
	filters, err := filterClient.GetFilters(ctx, &rolesrv.NilMessage{})

	if err != nil {
		return sendFatal(err.Error())
	}

	if len(filters.FilterList) == 0 {
		return sendError("No Filters\n")
	}

	buffer.WriteString("Filters:\n")
	for filter := range filters.FilterList {
		buffer.WriteString(fmt.Sprintf("\t%s: %s\n", filters.FilterList[filter].Name, filters.FilterList[filter].Description))
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

func removeRole(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return sendError("Usage: !role role_remove <role_name>")
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	roleClient := clientFactory.NewRoleClient()

	_, err = roleClient.RemoveRole(ctx, &rolesrv.Role{ShortName: req.Args[2]})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Removed: %s\n", req.Args[2]))
}

func removeFilter(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return sendError("Usage: !role filter_remove <filter_name>")
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	filterClient := clientFactory.NewFilterClient()

	_, err = filterClient.RemoveFilter(ctx, &rolesrv.Filter{Name: req.Args[2]})
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

func roleInfo(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) != 3 {
		return sendError("Usage: !role role_info <role_name>")
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	roleClient := clientFactory.NewRoleClient()

	info, err := roleClient.GetRole(ctx, &rolesrv.Role{ShortName: req.Args[2]})
	if err != nil {
		return sendFatal(err.Error())
	}

	return fmt.Sprintf("```ShortName: %s\nType: %s\nName: %s\nColor: %d\nHoist: %t\nPosition: %d\nPermissions: %d\nManaged: %t\nMentionable: %t\n```",
		info.ShortName, info.Type, info.Name, info.Color, info.Hoist, info.Position, info.Permissions, info.Managed, info.Mentionable)
}

//"member_list":   {listMembers, "List all Filter Members"},
//"member_add":    {addMember, "Add Filter Member"},
//"member_remove": {removeMember, "Remove Filter Member"},

func listMembers(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	if len(req.Args) != 3 {
		return sendError("Usage: !role member_list <filter_name>")
	}

	filterClient := clientFactory.NewFilterClient()
	members, err := filterClient.GetMembers(ctx, &rolesrv.Filter{Name: req.Args[2]})

	if err != nil {
		return sendFatal(err.Error())
	}

	if len(members.Members) == 0 {
		return sendError("No members in filter")
	}

	buffer.WriteString("Filter Members:\n")
	for member := range members.Members {
		buffer.WriteString(fmt.Sprintf("\t%s\n", members.Members[member]))
	}

	return fmt.Sprintf("```%s```", buffer.String())
}

func addMember(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 4 {
		return sendError("Usage: !role member_add <user> <filter>")
	}

	tmp := req.Args[2]
	user := tmp[2 : len(tmp)-1]
	filter := req.Args[3]

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	filterClient := clientFactory.NewFilterClient()

	_, err = filterClient.AddMembers(ctx,
		&rolesrv.Members{Name: []string{user}, Filter: filter})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Added '%s' to '%s'\n", user, filter))
}

func removeMember(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 4 {
		return sendError("Usage: !role remove_member <user> <filter>")
	}

	canPerform, err := canPerform(ctx, req, []string{"role_admins"})
	if err != nil {
		return sendFatal(err.Error())
	}

	if !canPerform {
		return sendError("User doesn't have permission to this command")
	}

	tmp := req.Args[2]
	user := tmp[2 : len(tmp)-1]
	filter := req.Args[3]

	filterClient := clientFactory.NewFilterClient()
	_, err = filterClient.RemoveMembers(ctx,
		&rolesrv.Members{Name: []string{user}, Filter: filter})
	if err != nil {
		return sendFatal(err.Error())
	}

	return sendSuccess(fmt.Sprintf("Removed '%s' from '%s'\n", user, filter))
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
