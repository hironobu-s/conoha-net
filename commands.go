package main

import (
	"fmt"
	"os"

	"sort"

	"bytes"

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
		Name:    "list-group",
		Aliases: []string{},
		Usage:   "List security groups",
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
		Usage:   "Create a security group",
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
		Name:    "update-group",
		Aliases: []string{},
		Usage:   "Update a security group",
		Action:  runCmd,
	},

	{
		Name:      "delete-group",
		Aliases:   []string{},
		Usage:     "Delete a security group",
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	{
		Name:    "create-rule",
		Aliases: []string{},
		Usage:   "Create rules",
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
		Name:    "list",
		Aliases: []string{},
		Usage:   "List VPS",
		Action:  runCmd,
	},

	{
		Name:    "attach",
		Aliases: []string{},
		Usage:   "Attach security groups to VPS",
		Flags: append(queryVpsFlags, cli.StringFlag{
			Name: "secgroup, s",
		}),
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},

	{
		Name:    "detach",
		Aliases: []string{},
		Usage:   "Dettach security groups from VPS",
		Flags: append(queryVpsFlags, cli.StringFlag{
			Name: "secgroup, s",
		}),
		ArgsUsage: "security-group-name",
		Action:    runCmd,
	},
}

var openstack *conoha.OpenStack

func runCmd(c *cli.Context) error {
	// Run
	switch c.Command.Name {
	case "create-rule":
		cmdCreateRule(c)

	case "list-group":
		cmdListGroup(c)
	case "create-group":
		cmdCreateGroup(c)
	case "delete-group":
		cmdDeleteGroup(c)

	case "list":
		cmdList(c)
	case "attach":
		cmdAttachOrDetach(c, "attach")
	case "detach":
		cmdAttachOrDetach(c, "detach")

	default:
		err := fmt.Errorf("Unimplemented command. [%s]", c.Command.Name)
		ExitOnError(err)
	}
	return nil
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

func cmdCreateRule(c *cli.Context) {
	var err error
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		ExitOnError(err)
	}

	rule := conoha.Rule{
		Direction:      c.String("direction"),
		EtherType:      c.String("ether-type"),
		PortRange:      c.String("port-range"),
		Protocol:       c.String("protocol"),
		RemoteGroupID:  c.String("remote-group-id"),
		RemoteIPPrefix: c.String("remote-ip-prefix"),
	}

	name := c.Args()[0]
	rt, err := conoha.CreateRule(openstack, name, rule)
	if err != nil {
		ExitOnError(err)
	}

	fmt.Printf("ID: %s created\n", rt.ID)
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

func cmdListGroup(c *cli.Context) {
	var err error
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		ExitOnError(err)
	}

	groups, err := conoha.ListGroup(openstack)
	if err != nil {
		ExitOnError(err)
	}
	if !c.Bool("all") {
		groups = conoha.RemoveSystemGroups(groups)
	}

	// Display
	var data DisplayData
	if len(groups) > 0 {
		data = append(data, []string{"Name", "Direction", "EtherType", "Proto", "IP Range", "Port"})
		for _, sg := range groups {
			for _, rule := range sg.Rules {
				cols := make([]string, 0, 6)
				cols = append(cols, sg.Name, rule.Direction, rule.EtherType)
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

	outputTable(data, true)
}

func cmdCreateGroup(c *cli.Context) {
	var err error
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		ExitOnError(err)
	}

	// description
	description := c.String("description")

	// security group name to create
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		ExitOnError(err)
	}
	name := c.Args()[0]

	if err = conoha.CreateGroup(openstack, name, description); err != nil {
		ExitOnError(err)
	}
}

func cmdDeleteGroup(c *cli.Context) {
	var err error
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		ExitOnError(err)
	}

	// security group name to delete
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		ExitOnError(err)
	}
	name := c.Args()[0]

	if err = conoha.DeleteGroup(openstack, name); err != nil {
		ExitOnError(err)
	}
}

func cmdList(c *cli.Context) {
	var err error
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		ExitOnError(err)
	}

	vpss, err := conoha.ListVps(openstack, nil)
	if err != nil {
		ExitOnError(err)
	}
	OutputVps(vpss)
}

func cmdAttachOrDetach(c *cli.Context, mode string) {
	var err error

	// security group name to attach or detach
	if c.NArg() == 0 {
		err = fmt.Errorf("Please specify the security group name")
		ExitOnError(err)
	}
	secGroup := c.Args()[0]

	// initialize openstack
	openstack, err = conoha.NewOpenStack()
	if err != nil {
		ExitOnError(err)
	}

	// detect vps to attach or detach
	vps, err := queryVps(c)
	if err != nil {
		ExitOnError(err)
	}

	// run
	if mode != "attach" && mode != "detach" {
		err = fmt.Errorf(`"mode" has to be either "attach" or "detach"`)
		ExitOnError(err)
	}

	if mode == "attach" {
		if err = conoha.Attach(openstack, vps, secGroup); err != nil {
			ExitOnError(err)
		}
	} else {
		if err = conoha.Detach(openstack, vps, secGroup); err != nil {
			ExitOnError(err)
		}
	}
}

func OutputVps(vpss []conoha.Vps) {
	numVps := len(vpss)
	data := make([][]string, 0, numVps)
	data = append(data, []string{"VPS", "NameTag", "SecGroup"})
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

		data = append(data, []string{vps.Ports[0].IPv4Address, vps.NameTag, buf.String()})
	}

	outputTable(data, true)
}

func outputTable(data [][]string, isFirstHeader bool) {
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
}
