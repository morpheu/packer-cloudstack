package cloudstack

import (
	"fmt"
	"github.com/morpheu/gopherstack"
	"log"
	"net/url"
)

type Artifact struct {
	// The name of the template
	templateName string

	// The ID of the image
	templateId string

	// ProjectId
	projectId string

	// The client for making API calls
	client *gopherstack.CloudstackClient
}

func (*Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Files() []string {
	// No local files created with Cloudstack.
	return nil
}

func (a *Artifact) Id() string {
	values := url.Values{}
	values.Set("templateid", a.templateId)
	values.Set("projecid", a.projectId)
	return a.client.BaseURL + "?" + values.Encode()
}

func (a *Artifact) String() string {
	return fmt.Sprintf("A template was created: UUID: %v - Name: %v",
		a.templateId, a.templateName)
}

func (a *Artifact) State(name string) interface{} {
	return nil
}

func (a *Artifact) Destroy() error {
	log.Printf("Delete template: %s", a.templateId)
	_, err := a.client.DeleteTemplate(a.templateId, a.projectId, "")
	return err
}
