// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-vsphere/builder/vsphere/driver"
)

type StepRemoteUpload struct {
	Datastore                  string
	Host                       string
	SetHostForDatastoreUploads bool
	RemoteCacheCleanup         bool
	UploadedCustomCD           bool
}

func (s *StepRemoteUpload) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	d := state.Get("driver").(driver.Driver)

	if path, ok := state.GetOk("iso_path"); ok {
		// user-supplied boot iso
		fullRemotePath, err := s.uploadFile(path.(string), d, ui)
		if err != nil {
			state.Put("error", err)
			return multistep.ActionHalt
		}
		state.Put("iso_remote_path", fullRemotePath)
	}
	if cdPath, ok := state.GetOk("cd_path"); ok {
		// Packer-created cd_files disk
		fullRemotePath, err := s.uploadFile(cdPath.(string), d, ui)
		if err != nil {
			state.Put("error", err)
			return multistep.ActionHalt
		}
		s.UploadedCustomCD = true
		state.Put("cd_path", fullRemotePath)
	}

	if s.RemoteCacheCleanup {
		state.Put("remote_cache_cleanup", s.RemoteCacheCleanup)
	}

	return multistep.ActionContinue
}

func GetRemoteDirectoryAndPath(path string, ds driver.Datastore) (string, string, string, string) {
	filename := filepath.Base(path)
	remotePath := fmt.Sprintf("packer_cache/%s", filename)
	remoteDirectory := fmt.Sprintf("[%s] packer_cache", ds.Name())
	fullRemotePath := fmt.Sprintf("%s/%s", remoteDirectory, filename)

	return filename, remotePath, remoteDirectory, fullRemotePath
}

func (s *StepRemoteUpload) uploadFile(path string, d driver.Driver, ui packersdk.Ui) (string, error) {
	ds, err := d.FindDatastore(s.Datastore, s.Host)
	if err != nil {
		return "", fmt.Errorf("error finding the datastore: %v", err)
	}

	filename, remotePath, remoteDirectory, fullRemotePath := GetRemoteDirectoryAndPath(path, ds)

	if exists := ds.FileExists(remotePath); exists == true {
		ui.Say(fmt.Sprintf("File %s already exists; skipping upload...", fullRemotePath))
		return fullRemotePath, nil
	}

	ui.Say(fmt.Sprintf("Uploading %s to %s...", filename, remoteDirectory))

	if exists := ds.DirExists(remotePath); exists == false {
		log.Printf("Cache directory does not exist; creating %s...", remoteDirectory)
		if err := ds.MakeDirectory(remoteDirectory); err != nil {
			return "", err
		}
	}

	if err := ds.UploadFile(path, remotePath, s.Host, s.SetHostForDatastoreUploads); err != nil {
		return "", err
	}
	return fullRemotePath, nil
}

func (s *StepRemoteUpload) Cleanup(state multistep.StateBag) {
	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	_, remoteCacheCleanup := state.GetOk("remote_cache_cleanup")

	if !cancelled && !halted && !remoteCacheCleanup {
		return
	}

	if !s.UploadedCustomCD {
		return
	}

	UploadedCDPath, ok := state.GetOk("cd_path")
	if !ok {
		return
	}

	ui := state.Get("ui").(packersdk.Ui)
	d := state.Get("driver").(*driver.VCenterDriver)
	ui.Say(fmt.Sprintf("Removing %s...", UploadedCDPath))

	ds, err := d.FindDatastore(s.Datastore, s.Host)
	if err != nil {
		log.Printf("Error finding the cache datastore. Please remove the item manually: %s", err)
		return
	}

	err = ds.Delete(UploadedCDPath.(string))
	if err != nil {
		log.Printf("Error removing item from the cache. Please remove the item manually: %s", err)
		return

	}
}
