// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package virtualbox

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	vbox "github.com/terra-farm/go-virtualbox"
)

func resourceNatNetwork() *schema.Resource {
	return &schema.Resource{
		Exists: resourceNatNetworkExists,
		Create: resourceNatNetworkCreate,
		Read:   resourceNatNetworkRead,
		Update: resourceNatNetworkUpdate,
		Delete: resourceNatNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"dhcp": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceNatNetworkExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	name := d.Get("name").(string)

	_, err := vbox.GetNATNetwork(name)
	if err != nil {
		return false, err
	}

	return true, nil
}

func resourceNatNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	dhcp := d.Get("dhcp").(bool)
	network := d.Get("network").(string)

	_, err := vbox.CreateNATNet(name, network, dhcp)
	if err != nil {
		return err
	}
	d.SetId(name)
	return resourceNatNetworkRead(d, meta)
}

func resourceNatNetworkRead(d *schema.ResourceData, meta interface{}) error {
	natnet, err := vbox.GetNATNetwork(d.Id())
	if err != nil {
		return err
	}
	d.Set("name", natnet.Name)
	d.Set("dhcp", natnet.DHCP)
	d.Set("network", natnet.IPv4.String())
	return nil
}

func resourceNatNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	natnet, err := vbox.GetNATNetwork(d.Id())
	if err != nil {
		return errLogf("unable to get nat network: %v", err)
	}
	if err := natnet.Config(); err != nil {
		return errLogf("unable to remove nat network: %v", err)
	}

	return nil
}

func resourceNatNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	natnet, err := vbox.GetNATNetwork(d.Id())
	if err != nil {
		return errLogf("unable to get nat network: %v", err)
	}
	if err := natnet.Delete(); err != nil {
		return errLogf("unable to remove nat network: %v", err)
	}
	return nil
}
