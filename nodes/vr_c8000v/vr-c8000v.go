// Copyright 2020 Nokia
// Licensed under the BSD 3-Clause License.
// SPDX-License-Identifier: BSD-3-Clause

package vr_c8000v

import (
	"fmt"
	"path"
	"regexp"

	"github.com/srl-labs/containerlab/nodes"
	"github.com/srl-labs/containerlab/types"
	"github.com/srl-labs/containerlab/utils"
)

var (
	kindNames          = []string{"cisco_c8000v"}
	defaultCredentials = nodes.NewCredentials("admin", "admin")

	InterfaceRegexp = regexp.MustCompile(`(?:Gi|GigabitEthernet)\s?(?P<port>\d+)$`)
	InterfaceOffset = 2
	InterfaceHelp   = "GiX or GigabitEthernetX (where X >= 2) or ethX (where X >= 1)"
)

const (
	scrapliPlatformName = "cisco_iosxe"

	generateable     = true
	generateIfFormat = "eth%d"
)

// Register registers the node in the NodeRegistry.
func Register(r *nodes.NodeRegistry) {
	generateNodeAttributes := nodes.NewGenerateNodeAttributes(generateable, generateIfFormat)
	platformAttrs := &nodes.PlatformAttrs{
		ScrapliPlatformName: scrapliPlatformName,
	}

	nrea := nodes.NewNodeRegistryEntryAttributes(defaultCredentials, generateNodeAttributes, platformAttrs)

	r.Register(kindNames, func() nodes.Node {
		return new(vrC8000v)
	}, nrea)
}

type vrC8000v struct {
	nodes.VRNode
}

func (n *vrC8000v) Init(cfg *types.NodeConfig, opts ...nodes.NodeOption) error {
	// Init VRNode
	n.VRNode = *nodes.NewVRNode(n, defaultCredentials, scrapliPlatformName)
	// set virtualization requirement
	n.HostRequirements.VirtRequired = true

	n.Cfg = cfg
	for _, o := range opts {
		o(n)
	}
	// env vars are used to set launch.py arguments in vrnetlab container
	defEnv := map[string]string{
		"CONNECTION_MODE":    nodes.VrDefConnMode,
		"USERNAME":           defaultCredentials.GetUsername(),
		"PASSWORD":           defaultCredentials.GetPassword(),
		"DOCKER_NET_V4_ADDR": n.Mgmt.IPv4Subnet,
		"DOCKER_NET_V6_ADDR": n.Mgmt.IPv6Subnet,
	}
	n.Cfg.Env = utils.MergeStringMaps(defEnv, n.Cfg.Env)

	// mount config dir to support startup-config functionality
	n.Cfg.Binds = append(n.Cfg.Binds, fmt.Sprint(path.Join(n.Cfg.LabDir, n.ConfigDirName), ":/config"))

	if n.Cfg.Env["CONNECTION_MODE"] == "macvtap" {
		// mount dev dir to enable macvtap
		n.Cfg.Binds = append(n.Cfg.Binds, "/dev:/dev")
	}

	n.Cfg.Cmd = fmt.Sprintf("--username %s --password %s --hostname %s --connection-mode %s --trace",
		n.Cfg.Env["USERNAME"], n.Cfg.Env["PASSWORD"], n.Cfg.ShortName, n.Cfg.Env["CONNECTION_MODE"])

	n.InterfaceRegexp = InterfaceRegexp
	n.InterfaceOffset = InterfaceOffset
	n.InterfaceHelp = InterfaceHelp

	return nil
}
