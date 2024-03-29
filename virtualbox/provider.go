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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Provider configuration structure
type Config struct {
	ReadyDelay   time.Duration
	ReadyTimeout time.Duration
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
			"ready_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"ready_timeout": &schema.Schema{
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
		ReadyDelay:   time.Duration(d.Get("ready_delay").(int)) * time.Second,
		ReadyTimeout: time.Duration(d.Get("ready_timeout").(int)) * time.Second,
	}

	if config.ReadyDelay == 0 {
		log.Printf("[INFO] No ready_delay was configured, using 60 seconds by default.")
		config.ReadyDelay = time.Duration(60) * time.Second
	}

	if config.ReadyTimeout == 0 {
		log.Printf("[INFO] No ready_timeout was configured, using 5 seconds by default.")
		config.ReadyTimeout = time.Duration(5) * time.Second
	}

	return config, nil
}
