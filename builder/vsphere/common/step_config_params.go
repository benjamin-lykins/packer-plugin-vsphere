// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type ConfigParamsConfig

package common

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/vim25/types"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/driver"
)

type ConfigParamsConfig struct {
	// Specifies a direct passthrough to the data object type that encapsulates configuration
	// settings when creating or reconfiguring a virtual machine. Refer to the vSphere API
	// documentation for the [`VirtualMachineConfigSpec`](https://developer.vmware.com/apis/vi-json/latest/data-structures/VirtualMachineConfigSpec/)
	// for available configuration parameters.
	ConfigParams map[string]string `mapstructure:"configuration_parameters"`
	// Specifies whether to enable time synchronization with the ESXi host where the virtual machine
	// is running. Defaults to `false`.
	ToolsSyncTime bool `mapstructure:"tools_sync_time"`
	// Specifies to automatically check for and upgrade VMware Tools following a virtual machine
	// power cycle if an upgrade is available. Defaults to `false`.
	ToolsUpgradePolicy bool `mapstructure:"tools_upgrade_policy"`
}

type StepConfigParams struct {
	Config *ConfigParamsConfig
}

func (s *StepConfigParams) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	vm := state.Get("vm").(*driver.VirtualMachineDriver)
	configParams := make(map[string]string)

	if s.Config.ConfigParams != nil {
		configParams = s.Config.ConfigParams
	}

	var info *types.ToolsConfigInfo
	if s.Config.ToolsSyncTime || s.Config.ToolsUpgradePolicy {
		info = &types.ToolsConfigInfo{}

		if s.Config.ToolsSyncTime {
			info.SyncTimeWithHost = &s.Config.ToolsSyncTime
		}

		if s.Config.ToolsUpgradePolicy {
			info.ToolsUpgradePolicy = "UpgradeAtPowerCycle"
		}
	}

	ui.Say("Adding configuration parameters...")
	if err := vm.AddConfigParams(configParams, info); err != nil {
		state.Put("error", fmt.Errorf("error adding configuration parameters: %v", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepConfigParams) Cleanup(state multistep.StateBag) {}
