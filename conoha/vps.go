package conoha

import (
	"fmt"
	"net"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/pagination"
)

type Vps struct {
	ID                  string
	NameTag             string
	ExternalIPv4Address net.IP
	ExternalIPv6Address net.IP
	ExternalPort        AttachedPort
	Ports               []AttachedPort
	SecurityGroups      []secgroups.SecurityGroup
}

type AttachedPort struct {
	PortId    string     `mapstructure:"port_id" json:"port_id"`
	PortState string     `mapstructure:"port_state" json:"port_state"`
	FixedIPs  []ports.IP `mapstructure:"fixed_ips" json:"fixed_ips"`
}

func (v *Vps) FromServer(s servers.Server) error {
	nametag, ok := s.Metadata["instance_name_tag"]
	if !ok {
		return fmt.Errorf("Attribute not found. [%s]", "instance_name_tag")
	}

	v.ID = s.ID
	v.NameTag = nametag.(string)
	if err := mapstructure.Decode(s.SecurityGroups, &v.SecurityGroups); err != nil {
		return err
	}

	for name, a := range s.Addresses {
		if !strings.HasPrefix(name, "ext-") {
			continue
		}
		addrs := a.([]interface{})
		for _, iaddr := range addrs {
			addr, ok := iaddr.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Can't convert to map[string]interface{}. [%v]", iaddr)
			}

			version, ok := addr["version"]
			if !ok {
				return fmt.Errorf(`Not has "version" field. [%v]`, addr)
			}

			straddr, ok := addr["addr"]
			if !ok {
				return fmt.Errorf(`Not has "addr" field. [%v]`, addr)
			}
			if version == 4.0 {
				v.ExternalIPv4Address = net.ParseIP(straddr.(string))
			} else if version == 6.0 {
				v.ExternalIPv6Address = net.ParseIP(straddr.(string))
			}
		}
	}

	return nil
}

// Set details of secutrity groups and ports
func (v *Vps) PopulateSecurityGroups(os *OpenStack) error {
	var err error

	// Security Groups
	result := servers.GetResult{}
	url := os.Compute.ServiceURL("servers", v.ID, "os-security-groups")
	_, err = os.Compute.Get(url, &result.Body, nil)
	if err != nil {
		return err
	}

	var resp struct {
		SecurityGroups []secgroups.SecurityGroup `mapstructure:"security_groups"`
	}

	if err = mapstructure.Decode(result.Body, &resp); err != nil {
		return err
	}
	v.SecurityGroups = resp.SecurityGroups

	return nil
}

func (v *Vps) PopulatePorts(os *OpenStack) error {
	var err error

	result := servers.GetResult{}
	url := os.Compute.ServiceURL("servers", v.ID, "os-interface")
	_, err = os.Compute.Get(url, &result.Body, nil)
	if err != nil {
		return err
	}

	var resp struct {
		Ports []AttachedPort `mapstructure:"interfaceAttachments" json:"ports"`
	}

	if err = mapstructure.Decode(result.Body, &resp); err != nil {
		return err
	}
	v.Ports = resp.Ports

	// Try to detect port that connect to global network.
	// In ConoHa, port that has IPv4 and Ipv6 addresses is it.

	_, p1, _ := net.ParseCIDR("10.0.0.0/8") // Private networks
	_, p2, _ := net.ParseCIDR("172.16.0.0/12")
	_, p3, _ := net.ParseCIDR("192.168.0.0/16")

	for _, p := range v.Ports {
		if p.PortState != "ACTIVE" {
			continue
		}

		var hasv4, hasv6 bool
		for _, fip := range p.FixedIPs {
			ip := net.ParseIP(fip.IPAddress)

			// Skip private address
			if p1.Contains(ip) || p2.Contains(ip) || p3.Contains(ip) {
				continue
			}

			if ip.To4() != nil {
				hasv4 = true
			} else {
				hasv6 = true
			}
		}
		if hasv4 && hasv6 {
			v.ExternalPort = p

			for _, fip := range p.FixedIPs {
				ip := net.ParseIP(fip.IPAddress)
				if ip.To4() != nil {
					v.ExternalIPv4Address = ip
				}
			}
		}
	}

	return nil
}

func (v *Vps) String() string {
	return fmt.Sprintf("%s %s %s", v.ID, v.NameTag, v.ExternalPort.FixedIPs[0].IPAddress)
}

func GetVps(os *OpenStack, query string) (*Vps, error) {
	query = strings.ToLower(query)

	condition := func(vps Vps) bool {
		if strings.ToLower(vps.ID) == query || strings.ToLower(vps.NameTag) == query {
			return true
		}
		return false
	}

	vpss, err := ListVps(os, condition)
	if err != nil {
		return nil, err
	} else if len(vpss) != 1 {
		return nil, nil
	} else {
		return &vpss[0], nil
	}
}

func ListVps(os *OpenStack, condition func(vps Vps) (match bool)) ([]Vps, error) {
	if condition == nil {
		condition = func(vps Vps) bool { return true }
	}

	var pager pagination.Pager

	opts := servers.ListOpts{}
	pager = servers.List(os.Compute, opts)

	vpss := make([]Vps, 0)
	err := pager.EachPage(func(pages pagination.Page) (bool, error) {
		ss, err := servers.ExtractServers(pages)
		if err != nil {
			return false, err
		}

		for _, s := range ss {
			vps := Vps{}
			if err := vps.FromServer(s); err != nil {
				return false, err
			}

			if condition(vps) {
				vpss = append(vpss, vps)
			}
		}
		return true, err
	})
	if err != nil {
		return nil, err
	}

	return vpss, nil
}
