// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package virtualbox serves as an entrypoint, returning the list of available
// resources for the plugin.
package virtualbox

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider configuration structure
type Config struct {
	Delay   int
	MinTimeout int
}

func init() {
	// Terraform is already adding the timestamp for us
	log.SetFlags(log.Lshortfile)
	log.SetPrefix(fmt.Sprintf("pid-%d-", os.Getpid()))
}

// Provider returns a resource provider for virtualbox.
func Provider() *schema.Provider {
	return &schema.Provider{
		// Provider configuration
		Schema: map[string]*schema.Schema{
			"delay": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"mintimeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"virtualbox_vm": resourceVM(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &Config{
		Delay:   d.Get("delay").(int),
		MinTimeout: d.Get("mintimeout").(int),
	}

	if config.Delay == 0 {
		log.Printf("[INFO] No Delay was configured, using 60 seconds by default.")
		config.Delay = 60
	}

	if config.MinTimeout == 0 {
		log.Printf("[INFO] No MinTimeout was configured, using 5 seconds by default.")
		config.MinTimeout = 5
	}

	return config, nil
}