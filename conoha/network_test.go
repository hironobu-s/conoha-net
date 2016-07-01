package conoha

import "github.com/mitchellh/mapstructure"
import "testing"
import "reflect"

func TestToCreateOpts(t *testing.T) {
	datasets := []map[string]string{
		{
			"SecurityGroupName": "test-name",
			"Direction":         "ingress",
			"EtherType":         "IPv6",
			"PortRange":         "80:8080",
			"Protocol":          "tcp",
			"RemoteGroupID":     "",
			"RemoteIPPrefix":    "192.168.0.0/24",
		},
		{
			"SecurityGroupName": "test-name",
			"Direction":         "egress",
			"EtherType":         "IPv4",
			"PortRange":         "80",
			"Protocol":          "udp",
			"RemoteGroupID":     "",
			"RemoteIPPrefix":    "192.168.0.0",
		},
	}

	for _, dataset := range datasets {
		r := RuleCreateOpts{}
		if err := mapstructure.Decode(dataset, &r); err != nil {
			t.Error(err)
		}

		secGroupName, opts, err := r.ToCreateOpts()
		if err != nil {
			t.Error(err)
		}

		if secGroupName != dataset["SecurityGroupName"] {
			t.Errorf(`"SecurityGroupName" name not match`)
		}

		for name, value := range dataset {
			if name == "SecurityGroupName" || name == "PortRange" {
				continue
			}

			v := reflect.ValueOf(opts)
			fieldValue := v.FieldByName(name).String()
			if fieldValue != value {
				t.Errorf(`"%s" name not match`, name)
			}
		}
	}
}
