// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package virtualbox serves as an entrypoint, returning the list of available
// resources for the plugin.
package provider

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terra-farm/go-virtualbox"
)

func init() {
	// Terraform is already adding the timestamp for us
	log.SetFlags(log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("pid-%d-", os.Getpid()))
}

// New returns a resource provider for virtualbox.
func New() *schema.Provider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"virtualbox_vm": resourceVM(),
		},
		ConfigureContextFunc: configure,
	}
}

// configure creates a new instance of the new virtualbox manager which will be
// used for communication with virtualbox.
func configure(context.Context, *schema.ResourceData) (any, diag.Diagnostics) {
	return virtualbox.NewManager(), nil
}
