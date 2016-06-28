package conoha

import (
	"os"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

type OpenStack struct {
	Compute *gophercloud.ServiceClient
	Network *gophercloud.ServiceClient
}

func NewOpenStack() (*OpenStack, error) {
	c, err := Compute()
	if err != nil {
		return nil, err
	}

	n, err := Network()
	if err != nil {
		return nil, err
	}

	cn := &OpenStack{
		Compute: c,
		Network: n,
	}

	return cn, nil
}

var identity *gophercloud.ProviderClient

func Identity() (*gophercloud.ProviderClient, error) {
	if identity == nil {
		// Credentials from env
		opts, err := openstack.AuthOptionsFromEnv()
		if err != nil {
			return nil, err
		}

		identity, err = openstack.AuthenticatedClient(opts)
		if err != nil {
			return nil, err
		}
	}
	return identity, nil
}

var _compute *gophercloud.ServiceClient

func Compute() (*gophercloud.ServiceClient, error) {
	if _compute == nil {
		client, err := Identity()
		if err != nil {
			return nil, err
		}

		eo := gophercloud.EndpointOpts{
			Type:   "compute",
			Region: os.Getenv("OS_REGION_NAME"),
		}

		_compute, err = openstack.NewComputeV2(client, eo)
		if err != nil {
			return nil, err
		}
	}
	return _compute, nil
}

var _network *gophercloud.ServiceClient

func Network() (*gophercloud.ServiceClient, error) {
	if _network == nil {
		client, err := Identity()
		if err != nil {
			return nil, err
		}

		// Endpoint options
		eo := gophercloud.EndpointOpts{
			Type:   "network",
			Region: os.Getenv("OS_REGION_NAME"),
		}

		// Set service client
		_network, err = openstack.NewNetworkV2(client, eo)
		if err != nil {
			return nil, err
		}
	}
	return _network, nil
}
