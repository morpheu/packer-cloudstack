package main

import (
	"github.com/mitchellh/packer/packer/plugin"
	"github.com/morpheu/packer-cloudstack"
)

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(cloudstack.Builder))
	server.Serve()
}
