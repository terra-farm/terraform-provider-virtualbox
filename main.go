// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/terra-farm/terraform-provider-virtualbox/virtualbox"
)

func main() {
	debug := flag.Bool("debug", false, "run the provider in debug mode")
	flag.Parse()

	opts := &plugin.ServeOpts{
		ProviderFunc: virtualbox.Provider,
	}

	if *debug {
		if err := plugin.Debug(
			context.Background(),
			"registry.terraform.io/terra-farm/virtualbox",
			opts,
		); err != nil {
			log.Fatalf("unable to run provider: %v", err)
		}
		return
	}

	plugin.Serve(opts)
}
