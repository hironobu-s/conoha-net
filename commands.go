package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"

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
		Flags: append(queryVpsFlags, cli.StringFlag{
			Name: "secgroup, s",
		}),
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

	fmt.Printf("ID: %s created\n", rt.ID)
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

type DisplayData [][]string

// Implements Sort interface to sort "Direction" column.
func (d DisplayData) Len() int {
	return len(d)
}

func (d DisplayData) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d DisplayData) Less(i, j int) bool {
	return d[i][1] < d[j][1]
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
	var data DisplayData
	if len(groups) > 0 {
		data = append(data, []string{"UUID", "SecurityGroup", "Direction", "EtherType", "Proto", "IP Range", "Port"})
		for _, sg := range groups {
			for _, rule := range sg.Rules {
				cols := make([]string, 0, 7)
				cols = append(cols, rule.ID, sg.Name, rule.Direction, rule.EtherType)
				if rule.Protocol != "" {
					cols = append(cols, rule.Protocol)
				} else {
					cols = append(cols, "ALL")
				}
				cols = append(cols, rule.RemoteIPPrefix)

				if rule.PortRangeMin == 0 && rule.PortRangeMax == 0 {
					cols = append(cols, "ALL")
				} else {
					cols = append(cols, fmt.Sprintf("%d - %d", rule.PortRangeMin, rule.PortRangeMax))
				}
				data = append(data, cols)
			}
		}

	} else {
		data = [][]string{[]string{"No security groups found"}}
	}

	sort.Sort(data)

	return outputTable(data, true)
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

	return conoha.CreateGroup(openstack, name, description)
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
	data = append(data, []string{"NameTag", "IPv4", "IPv6", "SecGroup"})
	for _, vps := range vpss {
		var buf bytes.Buffer
		var i = 0
		for _, sg := range vps.SecurityGroups {
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
	}

	return outputTable(data, true)
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
		if err = conoha.Attach(openstack, vps, secGroup); err != nil {
			goto ON_ERROR
		}
	} else {
		if err = conoha.Detach(openstack, vps, secGroup); err != nil {
			goto ON_ERROR
		}
	}
	return

ON_ERROR:
	return err
}

func outputTable(data [][]string, isFirstHeader bool) (err error) {
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
