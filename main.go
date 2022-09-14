// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package main

import (
	"flag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/terra-farm/terraform-provider-virtualbox/internal/provider"
)

func main() {
	debug := flag.Bool("debug", false, "run the provider in debug mode")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: provider.New,
		ProviderAddr: "registry.terraform.io/terra-farm/virtualbox",
		Debug:        *debug,
	})
}
