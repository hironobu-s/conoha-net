package conoha

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/pagination"
)

type Vps struct {
	ID             string
	NameTag        string
	Ports          []Port
	SecurityGroups []secgroups.SecurityGroup
}

func (v *Vps) String() string {
	return fmt.Sprintf("%s %s %s", v.ID, v.NameTag, v.Ports[0].IPv4Address)
}

func GetVps(os *OpenStack, query string) (*Vps, error) {
	query = strings.ToLower(query)

	condition := func(vps Vps) bool {
		if strings.ToLower(vps.ID) == query || strings.ToLower(vps.NameTag) == query {
			return true
		}

		for _, p := range vps.Ports {
			if p.IPv4Address == query || p.IPv6Address == query {
				return true
			}
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
			nametag, ok := s.Metadata["instance_name_tag"]
			if !ok {
				return false, fmt.Errorf("Attribute not found. [%s]", "instance_name_tag")
			}

			vps := Vps{
				ID:      s.ID,
				NameTag: nametag.(string),
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

	// Fetch security groups and rules
	var m sync.Mutex
	ret := make(chan error)

	for i, _ := range vpss {
		vps := &vpss[i]

		go func() {
			sg, err := vpsSecurityGroups(os, *vps)
			if err != nil {
				ret <- err
				return
			}
			m.Lock()
			vps.SecurityGroups = sg
			m.Unlock()

			ret <- nil
		}()

		go func() {
			ps, err := vpsPorts(os, *vps)
			if err != nil {
				ret <- err
				return
			}
			m.Lock()
			vps.Ports = ps
			m.Unlock()

			ret <- nil
		}()
	}

	var i = 0
	for i < len(vpss)*2 {
		if err = <-ret; err != nil {
			return nil, err
		}
		i++
	}

	return vpss, nil
}

func vpsSecurityGroups(os *OpenStack, vps Vps) ([]secgroups.SecurityGroup, error) {
	var err error

	result := servers.GetResult{}
	url := os.Compute.ServiceURL("servers", vps.ID, "os-security-groups")
	_, err = os.Compute.Get(url, &result.Body, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		SecGroups []secgroups.SecurityGroup `mapstructure:"security_groups" json:"security_groups"`
	}
	if err = mapstructure.Decode(result.Body, &resp); err != nil {
		return nil, err
	}
	return resp.SecGroups, nil
}

func vpsPorts(os *OpenStack, vps Vps) ([]Port, error) {
	result := servers.GetResult{}
	url := os.Compute.ServiceURL("servers", vps.ID, "os-interface")
	_, err := os.Compute.Get(url, &result.Body, nil)
	if err != nil {
		return nil, err
	}

	type OsPort struct {
		PortId    string     `mapstructure:"port_id" json:"port_id"`
		PortState string     `mapstructure:"port_state" json:"port_state"`
		FixedIPs  []ports.IP `mapstructure:"fixed_ips" json:"fixed_ips"`
	}

	var resp struct {
		OsPorts []OsPort `mapstructure:"interfaceAttachments" json:"ports"`
	}

	if err = mapstructure.Decode(result.Body, &resp); err != nil {
		panic(err)
		return nil, err
	}

	// Private networks
	_, p1, _ := net.ParseCIDR("10.0.0.0/8")
	_, p2, _ := net.ParseCIDR("172.16.0.0/12")
	_, p3, _ := net.ParseCIDR("192.168.0.0/16")

	ps := make([]Port, 0)
	for _, p := range resp.OsPorts {
		port := Port{PortId: p.PortId}

		if p.PortState != "ACTIVE" {
			continue
		}

		for _, fip := range p.FixedIPs {
			ip := net.ParseIP(fip.IPAddress)

			// Skip private address
			if p1.Contains(ip) || p2.Contains(ip) || p3.Contains(ip) {
				continue
			}

			if ip.To4() != nil {
				// Ipv4
				port.IPv4Address = ip.String()
			} else {
				// IPv6
				port.IPv6Address = ip.String()
			}
		}
		if port.IPv4Address != "" || port.IPv6Address != "" {
			ps = append(ps, port)
		}
	}

	return ps, nil
}
