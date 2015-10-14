package cloudstack

import (
	"errors"
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/morpheu/gopherstack"
	"log"
)

type stepCreateTemplate struct{
    snapshot_id string
}

func (s *stepCreateTemplate) Run(state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gopherstack.CloudstackClient)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(config)
	vmid := state.Get("virtual_machine_id").(string)
	osId := c.TemplateOSId

	ui.Say(fmt.Sprintf("Creating template: %v", c.TemplateName))

	if osId == "" {
		// get the volume id for the system volume for Virtual Machine 'id'
		listVmResponse, err := client.ListVirtualMachines(vmid, c.ProjectId, "")
		if err != nil {
			err := fmt.Errorf("Error creating template: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		// Check if the guest OS id is defined - if so, use that
		vmOsId := listVmResponse.Listvirtualmachinesresponse.Virtualmachine[0].Guestosid

		if vmOsId != "" {
			osId = vmOsId
		} else {
			// Fall back to default 103 (Other 64-Bit)
			osId = "103"
		}
	}

	// get the volume id for the system volume for Virtual Machine 'id'
	list_volumes, err := client.ListVolumes(vmid, c.ProjectId, "")
	if err != nil {
		err := fmt.Errorf("Error creating template: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// always use the first volume when creating a template
	rootId := list_volumes.Listvolumesresponse.Volume[0].ID

	// create snapshot for allow template export
	ui.Say("Creating snapshot from root disk...")
	create_snapshot, err := client.CreateSnapshot(rootId, c.ProjectId, "")
	if err != nil {
		err := fmt.Errorf("Error creating snapshot from root disk: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

    snapshot_jobid := create_snapshot.Createsnapshotresponse.JobId
	err = client.WaitForAsyncJob(snapshot_jobid, c.stateTimeout)
	if err != nil {
		err := fmt.Errorf("Error waiting for template to complete: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

    list_snapshots, err := client.ListSnapshots(rootId, c.ProjectId, "")
	s.snapshot_id = list_snapshots.Listsnapshotsresponse.Snapshot[0].ID

	createOpts := &gopherstack.CreateTemplate{
		Name:                  c.TemplateName,
		Displaytext:           c.TemplateDisplayText,
		Snapshotid:            s.snapshot_id,
		Ostypeid:              osId,
		Isdynamicallyscalable: c.TemplateScalable,
		Ispublic:              c.TemplatePublic,
		Isfeatured:            c.TemplateFeatured,
		Isextractable:         c.TemplateExtractable,
		Passwordenabled:       c.TemplatePasswordEnabled,
		ProjectId:             c.ProjectId,
		Account:               "",
	}

	response2, err := client.CreateTemplate(createOpts)
	if err != nil {
		err := fmt.Errorf("Error creating template: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Waiting for template to be saved...")
	jobid := response2.Createtemplateresponse.Jobid
	err = client.WaitForAsyncJob(jobid, c.stateTimeout)
	if err != nil {
		err := fmt.Errorf("Error waiting for template to complete: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	log.Printf("Looking up template ID for template: %s", c.TemplateName)
	response3, err := client.ListTemplates(c.TemplateName, "self", c.ProjectId, "")
	if err != nil {
		err := fmt.Errorf("Error looking up template ID: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Since if we create a template we should only have one with
	// that name, so we use the first response.
	template := response3.Listtemplatesresponse.Template[0].Name
	templateId := response3.Listtemplatesresponse.Template[0].ID

	if template != c.TemplateName {
		err := errors.New("Couldn't find template created. Bug?")
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("template_name", template)
	state.Put("template_id", templateId)

	return multistep.ActionContinue
}

func (s *stepCreateTemplate) Cleanup(state multistep.StateBag) {
    client := state.Get("client").(*gopherstack.CloudstackClient)
    ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(config)

    if s.snapshot_id == "" {
        return
    }

    ui.Say("Removing machine disk snapshot")
    response, err := client.DeleteSnapshot(s.snapshot_id, c.ProjectId, "")

	if err != nil {
		ui.Error(fmt.Sprintf(
			"Error removing snapshot. Please destroy it manually."))
	}

	status := response.Deletesnapshotresponse.Success
    ui.Say(fmt.Sprintf("Snapshot removal status: %s", status))

}
