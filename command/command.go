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
	NewFilterClient() rolesrv.FiltersClient
}

type command struct {
	funcptr func(ctx context.Context, request *proto.ExecRequest) string
	help    string
	args    []string
}

var cmdName = "role"
var commandList = map[string]command{
	"notDefined": {notDefined, "", []string{}},

	// Roles
	"role_list":   {listRoles, "List all Roles", []string{}},
	"role_add":    {addRole, "Add Role", []string{}},
	"role_remove": {removeRole, "Delete role", []string{}},
	"role_info":   {roleInfo, "Get Role Info", []string{}},
	"role_keys":   {roleKeys, "Get valid role keys", []string{}},
	"role_types":  {roleTypes, "Get valid role types", []string{}},
	"role_sync":   {syncRoles, "Sync Roles to chat service", []string{}},

	// Filters
	"filter_list":   {listFilters, "List all Filters", []string{}},
	"filter_add":    {addFilter, "Add Filter", []string{}},
	"filter_remove": {removeFilter, "Delete Filter", []string{}},
	"member_list":   {listMembers, "List all Filter Members", []string{}},
	"member_add":    {addMember, "Add Filter Member", []string{}},
	"member_remove": {removeMember, "Remove Filter Member", []string{}},
	"member_sync":   {syncMembers, "Sync Filter Membership", []string{}},
}

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

//
// Damn, this creates an initialization loop. Will need to figure out a better way to do this.
//
//func checkArgs(args []string, command string) error {
//	var buffer bytes.Buffer
//	argList := commandList[command].args
//
//	if len(args) < len(argList)+2 {
//		buffer.WriteString(fmt.Sprintf("Usage: !%s %s", cmdName, command))
//		for arg := range argList {
//			buffer.WriteString(fmt.Sprintf(" <%s>", arg))
//		}
//		return errors.New(buffer.String())
//	}
//
//	return nil
//}

func addRole(ctx context.Context, req *proto.ExecRequest) string {
	if len(req.Args) < 7 {
		return sendError("Usage: !role role_add <role_short_name> <role_type> <filterA> <filterB> <role_name>")
	}

	roleShortName := req.Args[2]
	roleType := req.Args[3]
	filterA := req.Args[4]
	filterB := req.Args[5]
	roleName := strings.Join(req.Args[6:], " ")

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
	_, err = roleClient.AddRole(ctx,
		&rolesrv.Role{
			ShortName: roleShortName,
			Type:      roleType,
			Name:      roleName,
			FilterA:   filterA,
			FilterB:   filterB,
		})

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

	return fmt.Sprintf("```ShortName: %s\nType: %s\nFilterA: %s\nFilterB: %s\nName: %s\nColor: %d\nHoist: %t\nPosition: %d\nPermissions: %d\nManaged: %t\nMentionable: %t\n```",
		info.ShortName,
		info.Type,
		info.FilterA,
		info.FilterB,
		info.Name,
		info.Color,
		info.Hoist,
		info.Position,
		info.Permissions,
		info.Managed,
		info.Mentionable,
	)
}

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

func syncRoles(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	roleClient := clientFactory.NewRoleClient()
	response, err := roleClient.SyncRoles(ctx, &rolesrv.NilMessage{})

	if err != nil {
		return sendFatal(err.Error())
	}

	if len(response.Added) == 0 {
		buffer.WriteString("No roles to add")
	} else {
		buffer.WriteString("Adding:\n")
		for r := range response.Added {
			buffer.WriteString(fmt.Sprintf("\t%s\n", response.Added[r]))
		}
	}

	if len(response.Removed) == 0 {
		buffer.WriteString("\nNo roles to remove")
	} else {
		buffer.WriteString("\nRemoving:\n")
		for r := range response.Removed {
			buffer.WriteString(fmt.Sprintf("\t%s\n", response.Removed[r]))
		}
	}

	return fmt.Sprintf("```%s\n```", buffer.String())
}

func syncMembers(ctx context.Context, req *proto.ExecRequest) string {
	var buffer bytes.Buffer
	roleClient := clientFactory.NewRoleClient()
	response, err := roleClient.SyncMembers(ctx, &rolesrv.NilMessage{})

	if err != nil {
		return sendFatal(err.Error())
	}

	if len(response.Added) == 0 {
		buffer.WriteString("No members to add")
	} else {
		buffer.WriteString("Adding:\n")
		for r := range response.Added {
			buffer.WriteString(fmt.Sprintf("\t%s\n", response.Added[r]))
		}
	}

	if len(response.Removed) == 0 {
		buffer.WriteString("\nNo members to remove")
	} else {
		buffer.WriteString("\nRemoving:\n")
		for r := range response.Removed {
			buffer.WriteString(fmt.Sprintf("\t%s\n", response.Removed[r]))
		}
	}

	return fmt.Sprintf("```%s\n```", buffer.String())
}

func notDefined(ctx context.Context, req *proto.ExecRequest) string {
	return "This command hasn't been defined yet"
}

func NewCommand(name string, factory ClientFactory) *Command {
	clientFactory = factory
	newCommand := Command{name: name, factory: factory}
	return &newCommand
}

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
