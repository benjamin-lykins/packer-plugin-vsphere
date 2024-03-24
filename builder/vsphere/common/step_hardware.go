// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown
//go:generate packer-sdc mapstructure-to-hcl2 -type HardwareConfig

package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/driver"
)

type HardwareConfig struct {
	// Specifies the number of virtual CPUs cores for the virtual machine.
	CPUs int32 `mapstructure:"CPUs"`
	// Specifies the number of virtual CPU cores per socket for the virtual machine.
	CpuCores int32 `mapstructure:"cpu_cores"`
	// Specifies the CPU reservation in MHz.
	CPUReservation int64 `mapstructure:"CPU_reservation"`
	// Specifies the upper limit of available CPU resources in MHz.
	CPULimit int64 `mapstructure:"CPU_limit"`
	// Specifies to enable CPU hot plug setting for virtual machine. Defaults to `false`
	CpuHotAddEnabled bool `mapstructure:"CPU_hot_plug"`
	// Specifies the amount of memory for the virtual machine in MB.
	RAM int64 `mapstructure:"RAM"`
	// Specifies the amount of reserved memory in MB.
	RAMReservation int64 `mapstructure:"RAM_reservation"`
	// Specifies to reserve all allocated memory. Defaults to `false`.
	//
	// -> **Note:** May not be used together with `RAM_reservation`.
	RAMReserveAll bool `mapstructure:"RAM_reserve_all"`
	// Specified to enable memory hot add setting for virtual machine. Defaults to `false`.
	MemoryHotAddEnabled bool `mapstructure:"RAM_hot_plug"`
	// Specifies the amount of video memory in KB. Defaults to 4096 KB.
	//
	// -> **Note:** Refer to the [vSphere documentation](https://docs.vmware.com/en/VMware-vSphere/8.0/vsphere-vm-administration/GUID-789C3913-1053-4850-A0F0-E29C3D32B6DA.html)
	// for supported maximums.
	VideoRAM int64 `mapstructure:"video_ram"`
	// Specifies the number of video displays. Defaults to `1`.
	//
	//`-> **Note:** Refer to the [vSphere documentation](https://docs.vmware.com/en/VMware-vSphere/8.0/vsphere-vm-administration/GUID-789C3913-1053-4850-A0F0-E29C3D32B6DA.html)
	// for supported maximums.
	Displays int32 `mapstructure:"displays"`
	// Specifies the vGPU profile for accelerated graphics. Defaults to `none`.
	//
	// -> **Note:** Refer to the [NVIDIA GRID vGPU documentation](https://docs.nvidia.com/grid/latest/grid-vgpu-user-guide/index.html#configure-vmware-vsphere-vm-with-vgpu)
	// for examples of profile names.
	VGPUProfile string `mapstructure:"vgpu_profile"`
	// Specified to enable nested hardware virtualization for the virtual machine. Defaults to
	// `false`.
	NestedHV bool `mapstructure:"NestedHV"`
	// Specifies the firmware for the virtual machine.
	//
	// The available options for this setting are: 'bios', 'efi', and 'efi-secure'.
	//
	// -> **Note:** Use `efi-secure` for UEFI Secure Boot.
	Firmware string `mapstructure:"firmware"`
	// Specifies to force entry into the BIOS setup screen during boot. Defaults to `false`.
	ForceBIOSSetup bool `mapstructure:"force_bios_setup"`
	// Specifies to enable virtual trusted platform module (TPM) device for the virtual machine.
	// Defaults to `false`.
	VTPMEnabled bool `mapstructure:"vTPM"`
	// Specifies the virtual precision clock device for the virtual machine. Defaults to `none`.
	//
	// The available options for this setting are: `none`, `ntp`, and `ptp`.
	VirtualPrecisionClock string `mapstructure:"precision_clock"`
}

func (c *HardwareConfig) Prepare() []error {
	var errs []error

	if c.RAMReservation > 0 && c.RAMReserveAll != false {
		errs = append(errs, fmt.Errorf("'RAM_reservation' and 'RAM_reserve_all' cannot be used together"))
	}

	if c.Firmware != "" && c.Firmware != "bios" && c.Firmware != "efi" && c.Firmware != "efi-secure" {
		errs = append(errs, fmt.Errorf("'firmware' must be '', 'bios', 'efi' or 'efi-secure'"))
	}

	if c.VTPMEnabled && c.Firmware != "efi" && c.Firmware != "efi-secure" {
		errs = append(errs, fmt.Errorf("'vTPM' could be enabled only when 'firmware' set to 'efi' or 'efi-secure'"))
	}

	if c.VirtualPrecisionClock != "" && c.VirtualPrecisionClock != "ptp" && c.VirtualPrecisionClock != "ntp" && c.VirtualPrecisionClock != "none" {
		errs = append(errs, fmt.Errorf("'precision_clock' must be '', 'ptp', 'ntp', or 'none'"))
	}

	return errs
}

type StepConfigureHardware struct {
	Config *HardwareConfig
}

func (s *StepConfigureHardware) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	vm := state.Get("vm").(driver.VirtualMachine)

	if *s.Config != (HardwareConfig{}) {
		ui.Say("Customizing hardware...")

		err := vm.Configure(&driver.HardwareConfig{
			CPUs:                  s.Config.CPUs,
			CpuCores:              s.Config.CpuCores,
			CPUReservation:        s.Config.CPUReservation,
			CPULimit:              s.Config.CPULimit,
			RAM:                   s.Config.RAM,
			RAMReservation:        s.Config.RAMReservation,
			RAMReserveAll:         s.Config.RAMReserveAll,
			NestedHV:              s.Config.NestedHV,
			CpuHotAddEnabled:      s.Config.CpuHotAddEnabled,
			MemoryHotAddEnabled:   s.Config.MemoryHotAddEnabled,
			VideoRAM:              s.Config.VideoRAM,
			Displays:              s.Config.Displays,
			VGPUProfile:           s.Config.VGPUProfile,
			Firmware:              s.Config.Firmware,
			ForceBIOSSetup:        s.Config.ForceBIOSSetup,
			VTPMEnabled:           s.Config.VTPMEnabled,
			VirtualPrecisionClock: s.Config.VirtualPrecisionClock,
		})
		if err != nil {
			state.Put("error", err)
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (s *StepConfigureHardware) Cleanup(multistep.StateBag) {}
