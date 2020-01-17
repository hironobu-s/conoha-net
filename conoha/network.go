package conoha

import (
	"fmt"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

// Specity the system created security groups and prefix
const (
	SYSTEM_SECGROUP_DEFAULT = "default"
	SYSTEM_SECGROUP_PREFIX  = "gncs"
)

type RuleCreateOpts struct {
	SecurityGroupName string
	Direction         string
	EtherType         string
	PortRange         string
	Protocol          string
	RemoteGroupID     string
	RemoteIPPrefix    string
}

// Convert conoha-net CreateOpts to gophercloud CreateOpts.
func (r *RuleCreateOpts) ToCreateOpts() (name string, opts rules.CreateOpts, err error) {
	if r.SecurityGroupName == "" {
		return name, opts, fmt.Errorf(`Must specify "security-group-name".`)
	}
	name = r.SecurityGroupName

	if r.Direction == "ingress" {
		opts.Direction = rules.DirIngress

	} else if  r.Direction == "egress" {
		opts.Direction = rules.DirEgress

	} else {
		return name, opts, fmt.Errorf(`"direction" must be either "ingress" or "egress"`)
	}

	if r.EtherType == "IPv4" {
		opts.EtherType = rules.EtherType4

	} else if r.EtherType == "IPv6" {
		opts.EtherType = rules.EtherType6

	} else {
		return name, opts, fmt.Errorf(`"ether-type" must be either "IPv4" or "IPv6"`)
	}

	if r.Protocol == "tcp" {
		opts.Protocol = rules.ProtocolTCP

	} else if r.Protocol == "udp" {
		opts.Protocol = rules.ProtocolUDP

	} else if r.Protocol == "icmp" {
		opts.Protocol = rules.ProtocolICMP

	} else if r.Protocol == "all" {
		opts.Protocol = ""

	} else {
		return name, opts, fmt.Errorf(`invalid protocol[%s]`, r.Protocol)
	}

	if r.PortRange != "" {
		m, err := regexp.MatchString(`^[0-9]+[\-:][0-9]+$`, r.PortRange)
		if err != nil {
			return name, opts, err

		} else if m {
			p := strings.Index(r.PortRange, ":")
			if p < 0 {
				p = strings.Index(r.PortRange, "-")
			}

			if p < 0 {
				// ないはず
				return name, opts, errors.New("invalid port range(may be a wrong regular expression)")
			}
			
			opts.PortRangeMin, err = strconv.Atoi(r.PortRange[:p])
			if err != nil {
				// ないはず
				return name, opts, errors.New("invalid port range(may be a wrong regular expression)")
			}
			opts.PortRangeMax, err = strconv.Atoi(r.PortRange[p+1:])
			if err != nil {
				// ないはず
				return name, opts, errors.New("invalid port range(may be a wrong regular expression)")
			}

		} else {
			p, err := strconv.Atoi(r.PortRange)
			if err != nil {
				return name, opts, fmt.Errorf("Invalid format of PortRange. [%s]", r.PortRange)
			}
			opts.PortRangeMin = p
			opts.PortRangeMax = p
		}

		// Must specify the protocol if port range is given.
		if opts.Protocol == "" {
			return name, opts, fmt.Errorf("Must specify the protocol if port range is given.")
		}
	}

	if r.RemoteGroupID != "" {
		opts.RemoteGroupID = r.RemoteGroupID
	}

	if r.RemoteIPPrefix != "" {
		opts.RemoteIPPrefix = r.RemoteIPPrefix
	}
	return name, opts, nil
}

// Create a security group rule and return created it.
func CreateRule(os *OpenStack, rule RuleCreateOpts) (*rules.SecGroupRule, error) {
	name, opts, err := rule.ToCreateOpts()
	if err != nil {
		return nil, err
	}

	// Detect the security group
	group, err := GetGroup(os, name)
	if err != nil {
		return nil, err
	}
	opts.SecGroupID = group.ID

	rt := rules.Create(os.Network, opts)
	if rt.Err != nil {
		return nil, rt.Err
	}
	return rt.Extract()
}

// Detele a security group rule
func DeleteRule(os *OpenStack, uuid string) error {
	return rules.Delete(os.Network, uuid).Err
}

// List the user created security groups.
func ListGroup(os *OpenStack) ([]groups.SecGroup, error) {
	opts := groups.ListOpts{}
	pager := groups.List(os.Network, opts)
	if pager.Err != nil {
		return nil, pager.Err
	}

	page, err := pager.AllPages()
	if err != nil {
		return nil, err
	}

	return groups.ExtractGroups(page)
}

// Return a security group
func GetGroup(os *OpenStack, name string) (*groups.SecGroup, error) {
	sgs, err := ListGroup(os)
	if err != nil {
		return nil, err
	}

	for _, g := range sgs {
		if g.ID == name || g.Name == name {
			return &g, nil
		}
	}

	return nil, fmt.Errorf("Can't found the security group. [%s]", name)
}

// Remove the system security groups from allgroups
func RemoveSystemGroups(allgroups []groups.SecGroup) []groups.SecGroup {
	ugs := make([]groups.SecGroup, 0, len(allgroups))
	for _, g := range allgroups {
		if g.Name != SYSTEM_SECGROUP_DEFAULT && !strings.HasPrefix(g.Name, SYSTEM_SECGROUP_PREFIX) {
			ugs = append(ugs, g)
		}
	}

	return ugs[0:len(ugs)]
}

// Create a security group
func CreateGroup(os *OpenStack, name string, description string) (*groups.SecGroup, error) {
	opts := groups.CreateOpts{
		Name:        name,
		Description: description,
	}
	return groups.Create(os.Network, opts).Extract()
}

// Delete a security group
func DeleteGroup(os *OpenStack, name string) error {
	group, err := GetGroup(os, name)
	if err != nil {
		return err
	}

	rt := groups.Delete(os.Network, group.ID)
	if rt.Err != nil {
		return rt.Err
	}
	return nil
}

// Attach security group to VPS and return attached security group.
//
// As for fixedIps or allowedAddressPairs,
// if those arguments are nil, current settings will be retained (will not be sent to API).
func Attach(os *OpenStack, vps *Vps, groupName string, fixedIps []string, allowedAddressPairs []string) (attached *groups.SecGroup, err error) {
	sgs, err := ListGroup(os)
	if err != nil {
		return nil, err
	}

	secGroupIds := make([]string, 0, len(vps.SecurityGroups))
	for _, g := range vps.SecurityGroups {
		secGroupIds = append(secGroupIds, g.ID)
	}

	for _, sg := range sgs {
		if sg.Name == groupName || sg.ID == groupName {
			attached = &sg
			secGroupIds = append(secGroupIds, sg.ID)
		}
	}
	if attached == nil {
		return nil, fmt.Errorf("Security group not found. [%s]", groupName)
	}

	opts := ports.UpdateOpts{
		SecurityGroups: &secGroupIds,
	}

	if fixedIps != nil {
		opts.FixedIPs = fixedIps
	}

	if allowedAddressPairs != nil {
		for _, ip := range allowedAddressPairs {
			*opts.AllowedAddressPairs = append(*opts.AllowedAddressPairs, ports.AddressPair{
				IPAddress: ip,
			})
		}
	}

	_, err = ports.Update(os.Network, vps.ExternalPort.PortId, opts).Extract()
	if err != nil {
		ed, ok := err.(gophercloud.ErrDefault400)
		if ok {
			err = errors.New(string(ed.Body))
		}
		return nil, err
	}

	return attached, nil
}

// Detach security group from VPS and return detached security group.
func Detach(os *OpenStack, vps *Vps, groupName string) (detached *secgroups.SecurityGroup, err error) {
	secGroupIds := make([]string, 0, len(vps.SecurityGroups))
	for _, sg := range vps.SecurityGroups {
		if sg.Name == groupName || sg.ID == groupName {
			detached = &sg
			continue
		} else {
			secGroupIds = append(secGroupIds, sg.ID)
		}
	}
	if detached == nil {
		return nil, fmt.Errorf("Security group not found. [%s]", groupName)
	}

	opts := ports.UpdateOpts{
		SecurityGroups: &secGroupIds,
	}
	_, err = ports.Update(os.Network, vps.ExternalPort.PortId, opts).Extract()
	if err != nil {
		ed, ok := err.(gophercloud.ErrDefault400)
		if ok {
			err = errors.New(string(ed.Body))
		}
		return nil, err
	}
	return detached, nil
}
