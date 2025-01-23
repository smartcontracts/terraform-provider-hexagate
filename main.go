package main

import (
	"github.com/smartcontracts/terraform-provider-hexagate/provider"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

var version = "dev"

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.New(version),
	})
}
