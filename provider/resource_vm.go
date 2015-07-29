// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package provider

import (
	"errors"
	"fmt"
	"log"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	vbox "github.com/riobard/go-virtualbox"
)

func resourceVM() *schema.Resource {
	return &schema.Resource{
		Exists: resourceVMExists,
		Create: resourceVMCreate,
		Read:   resourceVMRead,
		Update: resourceVMUpdate,
		Delete: resourceVMDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"cpus": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  "2",
			},

			"memory": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "512mb",
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_adapter": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "IntelPro1000MTServer",
						},
					},
				},
			},
		},
	}
}

type VM struct {
	vbox.Machine
	Image string
}

func resourceVMExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	uuid := d.Id()
	_, err := vbox.GetMachine(uuid)
	if err == nil {
		return true, nil
	} else if err == vbox.ErrMachineNotExist {
		return false, nil
	} else {
		return false, err
	}
}

func resourceVMCreate(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("name").(string)
	image := d.Get("image").(string)

	/* Get gold folder and machine folder */
	usr, err := user.Current()
	if err != nil {
		return err
	}
	goldFolder := filepath.Join(usr.HomeDir, ".terraform/virtualbox/gold")
	machineFolder := filepath.Join(usr.HomeDir, ".terraform/virtualbox/machine")

	/* Unpack gold image to gold folder */
	goldFileName := filepath.Base(image)
	goldName := strings.TrimSuffix(image, filepath.Ext(goldFileName))
	if filepath.Ext(goldName) == ".tar" {
		goldName = strings.TrimSuffix(goldName, ".tar")
	}
	goldPath := filepath.Join(goldFolder, goldName)
	unpackImage(image, goldPath)

	/* Gather all '.vdi' files */
	VDIs, err := filepath.Glob(filepath.Join(goldPath, "**.vdi"))
	if err != nil {
		return err
	}
	if len(VDIs) == 0 {
		return errors.New("No '.vdi' file found in gold image")
	}

	/* Create VM */
	vm, err := vbox.CreateMachine(name, machineFolder)
	if err != nil {
		log.Printf("[ERROR] Create virtualbox VM %s\n", vm.Name)
		return err
	}

	/* TODO: Copy the gold disk */

	/* TODO: Attach the disk to VM */

	/* Modify VM properties */
	err = vm.Modify()
	if err != nil {
		return err
	}

	/* Set ID */
	log.Printf("[DEBUG] Resource ID: %s\n", vm.UUID)
	d.SetId(vm.UUID)

	/* TODO: Set connection info */
	// d.SetConnInfo(map[string]string{
	// 	"type": "ssh",
	// 	"host": vm.IPAddress,
	// })

	return resourceVMRead(d, meta)
}

func resourceVMRead(d *schema.ResourceData, meta interface{}) error {
	/* Find VM */
	uuid := d.Id()
	vm, err := vbox.GetMachine(uuid)
	if err != nil {
		return err
	}

	/* This is to let TF know the resource is gone */
	if vm.State != vbox.Running {
		return nil
	}

	/* Refreshes only what makes sense, for example, we do not refresh settings
	that modify the behavior of this provider */
	// d.Set("name", vm.Name)
	// d.Set("cpus", vm.CPUs)
	// d.Set("memory", vm.Memory)
	// d.Set("ip_address", vm.IPAddress)

	// err = net_vix_to_tf(vm, d)
	// if err != nil {
	// 	return err
	// }

	// err = cdrom_vix_to_tf(vm, d)
	// if err != nil {
	// 	return err
	// }

	return nil
}

func resourceVMUpdate(d *schema.ResourceData, meta interface{}) error {
	// vm := new(vbox.Machine)

	/* Stop VM */

	/* Modify VM */

	// // Maps terraform.ResourceState attrbutes to vbox.Machine
	// tf_to_vbox(d, vm)

	// err := vm.Modify()
	// if err != nil {
	// 	return err
	// }

	/* Start VM */

	return resourceVMRead(d, meta)
}

func resourceVMDelete(d *schema.ResourceData, meta interface{}) error {
	uuid := d.Id()
	vm, err := vbox.GetMachine(uuid)
	if err != nil {
		return err
	}

	return vm.Delete()
}

func tf_to_vbox_net_device(attr string) (vbox.NICHardware, error) {
	switch attr {
	case "PCIII":
		return vbox.AMDPCNetPCIII, nil
	case "FASTIII":
		return vbox.AMDPCNetFASTIII, nil
	case "IntelPro1000MTDesktop":
		return vbox.IntelPro1000MTDesktop, nil
	case "IntelPro1000TServer":
		return vbox.IntelPro1000TServer, nil
	case "IntelPro1000MTServer":
		return vbox.IntelPro1000MTServer, nil
	default:
		return "", fmt.Errorf("[ERROR] Invalid virtual network device: %s", attr)
	}
}

func tf_to_vbox_network_type(attr string) (vbox.NICNetwork, error) {
	switch attr {
	case "bridged":
		return vbox.NICNetBridged, nil
	case "nat":
		return vbox.NICNetNAT, nil
	case "hostonly":
		return vbox.NICNetHostonly, nil
	case "internal":
		return vbox.NICNetInternal, nil
	case "generic":
		return vbox.NICNetGeneric, nil
	default:
		return "", fmt.Errorf("[ERROR] Invalid virtual network adapter type: %s", attr)
	}
}

// Maps Terraform attributes to provider's structs
// func tf_to_vbox(d *schema.ResourceData, vm *vbox.Machine) error {
// 	var err error

// 	vm.Name = d.Get("name").(string)
// 	vm.CPUs = uint(d.Get("cpus").(int))
// 	vm.Memory = uint(d.Get("memory").(int))

// 	// Maps any defined networks to VIX provider's data types
// 	err = net_tf_to_vbox(d, vm)
// 	if err != nil {
// 		return fmt.Errorf("Error mapping TF network adapter resource to VIX data types: %s", err)
// 	}

// 	if i := d.Get("image.#").(int); i > 0 {
// 		prefix := "image.0."
// 		vm.Image = vix.Image{
// 			URL: d.Get(prefix + "url").(string),
// 		}
// 	}

// 	return nil
// }

// func net_vbox_to_tf(vm *vbox.Machine, d *schema.ResourceData) error {

// 	vix_to_tf_network_type := func(netType vbox.NetworkType) string {
// 		switch netType {
// 		case vbox.NETWORK_CUSTOM:
// 			return "custom"
// 		case vbox.NETWORK_BRIDGED:
// 			return "bridged"
// 		case vbox.NETWORK_HOSTONLY:
// 			return "hostonly"
// 		case vbox.NETWORK_NAT:
// 			return "nat"
// 		default:
// 			return ""
// 		}
// 	}

// 	vix_to_tf_macaddress := func(adapter *vbox.NetworkAdapter) string {
// 		static := adapter.MacAddress.String()
// 		generated := adapter.GeneratedMacAddress.String()

// 		if static != "" {
// 			return static
// 		}

// 		return generated
// 	}

// 	vix_to_tf_vdevice := func(vdevice vbox.VNetDevice) string {
// 		switch vdevice {
// 		case vbox.NETWORK_DEVICE_E1000:
// 			return "e1000"
// 		case vbox.NETWORK_DEVICE_VLANCE:
// 			return "vlance"
// 		case vbox.NETWORK_DEVICE_VMXNET3:
// 			return "vmxnet3"
// 		default:
// 			return ""
// 		}
// 	}

// 	numvnics := len(vm.VNetworkAdapters)
// 	if numvnics <= 0 {
// 		return nil
// 	}

// 	prefix := "network_adapter"

// 	d.Set(prefix+".#", strconv.Itoa(numvnics))
// 	for i, adapter := range vm.VNetworkAdapters {
// 		attr := fmt.Sprintf("%s.%d.", prefix, i)
// 		d.Set(attr+"type", vix_to_tf_network_type(adapter.ConnType))
// 		d.Set(attr+"mac_address", vix_to_tf_macaddress(adapter))
// 		if adapter.ConnType == vbox.NETWORK_CUSTOM {
// 			d.Set(attr+"vswitch", "TODO(c4milo)")
// 		}
// 		d.Set(attr+"driver", vix_to_tf_vdevice(adapter.Vdevice))
// 	}

// 	return nil
// }
