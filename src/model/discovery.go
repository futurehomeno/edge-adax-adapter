package model

import (
	"github.com/futurehomeno/fimpgo/discovery"
)

func GetDiscoveryResource() discovery.Resource {
	return discovery.Resource{
		ResourceName:           ServiceName,
		ResourceType:           discovery.ResourceTypeAd,
		Author:                 "your email",
		IsInstanceConfigurable: false,
		InstanceId:             "1",
		Version:                "1",
		AdapterInfo: discovery.AdapterInfo{
			Technology:            "adax",
			FwVersion:             "all",
			NetworkManagementType: "inclusion_exclusion",
			// Services:              nil,   // Services must be defines in manifest
		},
	}

}
