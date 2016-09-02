package main

import (
	"bytes"
	"fmt"
	"os"

	"encoding/json"

	"strings"

	"github.com/hironobu-s/conoha-net/conoha"
	"github.com/urfave/cli"
)

var queryVpsFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "name, n",
		Value: "",
		Usage: "VPS name",
	},
	cli.StringFlag{
		Name:  "ip, i",
		Value: "",
		Usage: "VPS IP address",
	},
	cli.StringFlag{
		Name:  "id",
		Value: "",
		Usage: "VPS UUID",
	},
}

var commands = []cli.Command{
	{
		Name:    "list",
		Aliases: []string{},
		Usage:   "list all VPS",
		Action:  runCmd,
	},

	{
		Name:    "attach",
		Aliases: []string{},
		Usage:   "attach a security group to VPS",
		Flags: append(queryVpsFlags,
			cli.StringFlag{
				Name:  "secgroup, s",
				Usage: "Security group name",
			},
			cli.StringFlag{
				Name:   "fixed-ips, f",
				Hidden: true,
			},
			cli.StringFlag{
				Name:   "allowed-address-pairs, p",
				Hidden: true,
			},
		),
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	{
		Name:    "detach",
		Aliases: []string{},
		Usage:   "dettach a security group from VPS",
		Flags: append(queryVpsFlags, cli.StringFlag{
			Name: "secgroup, s",
		}),
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	// ---------

	{
		Name:    "list-group",
		Aliases: []string{},
		Usage:   "list security groups and rules",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "all,a",
				Usage: "List all security groups (including system groups).",
			},
		},
		Action: runCmd,
	},

	{
		Name:    "create-group",
		Aliases: []string{},
		Usage:   "create a security group",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "description,d",
				Usage: "Description of security group",
			},
		},
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	{
		Name:      "delete-group",
		Aliases:   []string{},
		Usage:     "delete a security group",
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	{
		Name:    "create-rule",
		Aliases: []string{},
		Usage:   "create a security group rule",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "d,direction",
				Usage: `(Required) The direction in which the rule applied. Must be either "ingress" or "egress"`,
				Value: "ingress",
			},

			cli.StringFlag{
				Name:  "e,ether-type",
				Usage: `(Required) Type of IP version. Must be either "Ipv4" or "Ipv6".`,
				Value: "IPv4",
			},

			cli.StringFlag{
				Name:  "p,port-range",
				Usage: ` The source port or port range. For example "80", "80-8080".`,
			},

			cli.StringFlag{
				Name:  "P,protocol",
				Usage: ` The IP protocol. Valid value are "tcp", "udp", "icmp" or "all".`,
				Value: "all",
			},

			cli.StringFlag{
				Name:  "g,remote-group-id",
				Usage: ` The remote group ID to be associated with this rule.`,
			},

			cli.StringFlag{
				Name:  "i,remote-ip-prefix",
				Usage: ` The IP prefix to be associated with this rule.`,
			},
		},
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	{
		Name:      "delete-rule",
		Aliases:   []string{},
		Usage:     "delete a security group rule",
		ArgsUsage: "uuid-of-rule",
		Action:    runCmd,
	},
}

var openstack *conoha.OpenStack

func runCmd(c *cli.Context) (err error) {
	// Run
	switch c.Command.Name {
	case "create-rule":
		err = cmdCreateRule(c)
	case "delete-rule":
		err = cmdDeleteRule(c)

	case "list-group":
		err = cmdListGroup(c)
	case "create-group":
		err = cmdCreateGroup(c)
	case "delete-group":
		err = cmdDeleteGroup(c)

	case "list":
		err = cmdList(c)
	case "attach":
		err = cmdAttachOrDetach(c, "attach")
	case "detach":
		err = cmdAttachOrDetach(c, "detach")

	default:
		return fmt.Errorf("Unimplemented command. [%s]", c.Command.Name)
	}
	return err
}

func queryVps(c *cli.Context) (*conoha.Vps, error) {
	var query string
	query = c.String("name")
	if query == "" {
		query = c.String("ip")
	}
	if query == "" {
		query = c.String("id")
	}
	if query == "" {
		return nil, fmt.Errorf("%s", `Choose at least one of "name", "ip" or "id" option to detect VPS.`)
	}

	vps, err := conoha.GetVps(openstack, query)
	if err != nil {
		return nil, err
	} else if vps == nil {
		return nil, fmt.Errorf("%s", "VPS not found")
	} else {
		return vps, nil
	}
}

func cmdCreateRule(c *cli.Context) (err error) {
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		return err
	}

	var name string
	if len(c.Args()) > 0 {
		name = c.Args()[0]
	} else {
		name = ""
	}
	rule := conoha.RuleCreateOpts{
		SecurityGroupName: name,
		Direction:         c.String("direction"),
		EtherType:         c.String("ether-type"),
		PortRange:         c.String("port-range"),
		Protocol:          c.String("protocol"),
		RemoteGroupID:     c.String("remote-group-id"),
		RemoteIPPrefix:    c.String("remote-ip-prefix"),
	}

	rt, err := conoha.CreateRule(openstack, rule)
	if err != nil {
		return err
	}

	if c.GlobalString("output") == "json" {
		return outputJson(map[string]string{"uuid": rt.ID})
	} else {
		return outputTable([][]string{[]string{rt.ID}})
	}

	return nil
}

func cmdDeleteRule(c *cli.Context) (err error) {
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		return err
	}

	// uuid of rule to delete
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		return err
	}
	uuid := c.Args()[0]

	return conoha.DeleteRule(openstack, uuid)
}

func cmdListGroup(c *cli.Context) (err error) {
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		return err
	}

	groups, err := conoha.ListGroup(openstack)
	if err != nil {
		return err
	}
	if !c.Bool("all") {
		groups = conoha.RemoveSystemGroups(groups)
	}

	// Display
	data := make([][]string, 0, len(groups))
	jsondata := make([]map[string]interface{}, 0, len(groups))

	if len(groups) > 0 {
		data = append(data, []string{"UUID", "SecurityGroup", "Direction", "EtherType", "Proto", "IP Range", "Port"})
		for _, sg := range groups {
			for _, rule := range sg.Rules {
				cols := make([]string, 0, 7)
				cols = append(cols, rule.ID, sg.Name, rule.Direction, rule.EtherType)

				jsoncols := map[string]interface{}{
					"uuid":           rule.ID,
					"security-group": sg.Name,
					"direction":      rule.Direction,
					"ether-type":     rule.EtherType,
					"proto":          "",
					"ip-range":       "",
					"port":           "",
				}

				if rule.Protocol != "" {
					cols = append(cols, rule.Protocol)
					jsoncols["proto"] = rule.Protocol
				} else {
					cols = append(cols, "ALL")
					jsoncols["proto"] = "ALL"
				}
				cols = append(cols, rule.RemoteIPPrefix)
				jsoncols["ip-range"] = rule.RemoteIPPrefix

				if rule.PortRangeMin == 0 && rule.PortRangeMax == 0 {
					cols = append(cols, "ALL")
					jsoncols["port"] = "ALL"
				} else {
					fmt.Sprintf("%d - %d", rule.PortRangeMin, rule.PortRangeMax)
					cols = append(cols, fmt.Sprintf("%d - %d", rule.PortRangeMin, rule.PortRangeMax))
					jsoncols["port"] = map[string]int{
						"min": rule.PortRangeMin,
						"max": rule.PortRangeMax,
					}
				}
				data = append(data, cols)
				jsondata = append(jsondata, jsoncols)
			}
		}

	} else {
		data = append(data, []string{"No security groups found"})
	}

	if c.GlobalString("output") == "json" {
		return outputJson(jsondata)
	} else {
		return outputTable(data)
	}
}

func cmdCreateGroup(c *cli.Context) (err error) {
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		return err
	}

	// description
	description := c.String("description")

	// security group name to create
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		return err
	}
	name := c.Args()[0]

	created, err := conoha.CreateGroup(openstack, name, description)
	if err != nil {
		return err
	}

	if c.GlobalString("output") == "json" {
		return outputJson(map[string]string{"uuid": created.ID})
	} else {
		return outputTable([][]string{[]string{created.ID}})
	}
}

func cmdDeleteGroup(c *cli.Context) (err error) {
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		return err
	}

	// security group name to delete
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		return err
	}
	name := c.Args()[0]

	return conoha.DeleteGroup(openstack, name)
}

func cmdList(c *cli.Context) (err error) {
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		return err
	}

	vpss, err := conoha.ListVps(openstack, nil)
	if err != nil {
		return err
	}

	numVps := len(vpss)
	data := make([][]string, 0, numVps)
	jsondata := make([]map[string]interface{}, 0, numVps)

	data = append(data, []string{"NameTag", "IPv4", "IPv6", "SecurityGroups"})
	for _, vps := range vpss {
		var buf bytes.Buffer
		var i = 0
		sgs := make([]string, 0, len(vps.SecurityGroups))
		for _, sg := range vps.SecurityGroups {
			sgs = append(sgs, sg.Name)
			buf.WriteString(sg.Name)
			i++
			if len(vps.SecurityGroups) != i {
				buf.WriteString(", ")
			}
		}

		data = append(data, []string{
			vps.NameTag,
			vps.ExternalIPv4Address.String(),
			vps.ExternalIPv6Address.String(),
			buf.String(),
		})

		jsondata = append(jsondata, map[string]interface{}{
			"name-tag":        vps.NameTag,
			"ipv4":            vps.ExternalIPv4Address.String(),
			"ipv6":            vps.ExternalIPv6Address.String(),
			"security-groups": sgs,
		})
	}

	if c.GlobalString("output") == "json" {
		return outputJson(jsondata)
	} else {
		return outputTable(data)
	}
}

func cmdAttachOrDetach(c *cli.Context, mode string) (err error) {
	var vps *conoha.Vps
	var secGroup string

	// security group name to attach or detach
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		goto ON_ERROR
	}
	secGroup = c.Args()[0]

	// initialize openstack
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		goto ON_ERROR
	}

	// detect vps to attach or detach
	vps, err = queryVps(c)
	if err != nil {
		goto ON_ERROR
	}

	// fetch details of port and security groups
	if err := vps.PopulateSecurityGroups(openstack); err != nil {
		goto ON_ERROR
	}
	if err := vps.PopulatePorts(openstack); err != nil {
		goto ON_ERROR
	}

	// run
	if mode != "attach" && mode != "detach" {
		err = fmt.Errorf(`"mode" has to be either "attach" or "detach"`)
		goto ON_ERROR
	}

	if mode == "attach" {
		var fixedIps, allowedAddressPairs []string = nil, nil

		if c.IsSet("fixed-ips") {
			fixedIps = strings.Split(c.String("fixed-ips"), ",")
		}

		if c.IsSet("allowed-address-pairs") {
			allowedAddressPairs = strings.Split(c.String("allowed-address-pairs"), ",")
		}

		attached, err := conoha.Attach(openstack, vps, secGroup, fixedIps, allowedAddressPairs)
		if err != nil {
			goto ON_ERROR
		}

		if c.GlobalString("output") == "json" {
			return outputJson(map[string]interface{}{"uuid": attached.ID})
		} else {
			return outputTable([][]string{[]string{attached.ID}})
		}

	} else {
		detached, err := conoha.Detach(openstack, vps, secGroup)
		if err != nil {
			goto ON_ERROR
		}
		if c.GlobalString("output") == "json" {
			return outputJson(map[string]interface{}{"uuid": detached.ID})
		} else {
			return outputTable([][]string{[]string{detached.ID}})
		}
	}

ON_ERROR:
	return err
}

func outputJson(data interface{}) error {
	strjson, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Printf("%s", strjson)
	return nil
}

func outputTable(data [][]string) (err error) {
	if len(data) == 0 {
		return
	}

	colLen := make([]int, len(data[0]))
	for _, row := range data {
		for j, col := range row {
			if len(col) > colLen[j] {
				colLen[j] = len(col)
			}
		}
	}

	for _, row := range data {
		l := len(row)
		for j, col := range row {
			fmt.Fprintf(os.Stdout, fmt.Sprintf("%%-%ds", colLen[j]), col)
			if j != l-1 {
				fmt.Fprintf(os.Stdout, "     ")
			}
		}
		fmt.Fprintf(os.Stdout, "\n")
	}
	return nil
}
