package provider

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/hashicorp/terraform/helper/resource"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	vbox "github.com/ccll/go-virtualbox"
	"github.com/hashicorp/terraform/helper/schema"
)

func init() {
	vbox.Verbose = true
}

func resourceVM() *schema.Resource {
	return &schema.Resource{
		// Exists: resourceVMExists,
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

			"image": &schema.Schema{
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
				Default:  "512mib",
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "IntelPro1000MTServer",
						},
						"host_interface": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"status": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4_address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4_address_available": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
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
	log.Printf("============ Exists")
	return false, nil
	uuid := d.Id()
	_, err := vbox.GetMachine(uuid)
	log.Printf("SHIT FUCK")
	if err == nil {
		return true, nil
	} else if err == vbox.ErrMachineNotExist {
		return false, nil
	} else {
		return false, err
	}
}

func resourceVMCreate(d *schema.ResourceData, meta interface{}) error {
	/* TODO: allow partial updates */

	image := d.Get("image").(string)

	/* Get gold folder and machine folder */
	usr, err := user.Current()
	if err != nil {
		log.Printf("[ERROR] Get current user: %s", err.Error())
		return err
	}
	goldFolder := filepath.Join(usr.HomeDir, ".terraform/virtualbox/gold")
	machineFolder := filepath.Join(usr.HomeDir, ".terraform/virtualbox/machine")
	os.MkdirAll(goldFolder, 0740)
	os.MkdirAll(machineFolder, 0740)

	/* Unpack gold image to gold folder */
	goldFileName := filepath.Base(image)
	goldName := strings.TrimSuffix(goldFileName, filepath.Ext(goldFileName))
	if filepath.Ext(goldName) == ".tar" {
		goldName = strings.TrimSuffix(goldName, ".tar")
	}
	goldPath := filepath.Join(goldFolder, goldName)
	err = unpackImage(image, goldPath)
	if err != nil {
		log.Printf("[ERROR] Unpack image %s: %s", image, err.Error())
		return err
	}

	/* Gather '*.vdi' and "*.vmdk" files from gold */
	goldDisks, err := gatherDisks(goldPath)
	if err != nil {
		return err
	}

	/* Create VM instance */
	name := d.Get("name").(string)
	vm, err := vbox.CreateMachine(name, machineFolder)
	if err != nil {
		log.Printf("[ERROR] Create virtualbox VM %s: %s\n", name, err.Error())
		return err
	}

	/* Clone gold virtual disk files to VM folder */
	for _, src := range goldDisks {
		filename := filepath.Base(src)
		target := filepath.Join(vm.BaseFolder, filename)
		err = vbox.CloneHD(src, target)
		if err != nil {
			log.Printf("[ERROR] Clone *.vdi and *.vmdk to VM folder: %s", err.Error())
			return err
		}
	}

	/* Attach virtual disks to VM */
	vmDisks, err := gatherDisks(vm.BaseFolder)
	if err != nil {
		return err
	}
	err = vm.AddStorageCtl("SATA", vbox.StorageController{
		SysBus:      vbox.SysBusSATA,
		Ports:       uint(len(vmDisks)) + 1,
		Chipset:     vbox.CtrlIntelAHCI,
		HostIOCache: true,
		Bootable:    true,
	})
	if err != nil {
		log.Printf("[ERROR] Create VirtualBox storage controller: %s", err.Error())
		return err
	}
	for i, disk := range vmDisks {
		err = vm.AttachStorage("SATA", vbox.StorageMedium{
			Port:      uint(i),
			Device:    0,
			DriveType: vbox.DriveHDD,
			Medium:    disk,
		})
		if err != nil {
			log.Printf("[ERROR] Attach VirtualBox storage medium: %s", err.Error())
			return err
		}
	}

	/* Setup VM general properties */
	err = tf_to_vbox(d, vm)
	if err != nil {
		log.Printf("[ERROR] Convert Terraform data to VM properties: %s", err.Error())
		return err
	}
	err = vm.Modify()
	if err != nil {
		log.Printf("[ERROR] Setup VM properties: %s", err.Error())
		return err
	}

	/* Start the VM */
	err = vm.Start()
	if err != nil {
		log.Printf("[ERROR] Start VM: %s", err.Error())
		return err
	}

	/* Assign VM ID */
	log.Printf("[DEBUG] Resource ID: %s\n", vm.UUID)
	d.SetId(vm.UUID)

	err = WaitUntilVMIsReady(d, vm, meta)
	if err != nil {
		log.Printf("[ERROR] Wait VM unitl ready: %s", err.Error())
		return err
	}

	return resourceVMRead(d, meta)
}

func resourceVMRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("============ Read")
	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		/* VM no longer exist */
		if err == vbox.ErrMachineNotExist {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", vm.Name)
	d.Set("cpus", vm.CPUs)
	bytes := uint64(vm.Memory) * humanize.MiByte
	repr := humanize.IBytes(bytes)
	d.Set("memory", strings.ToLower(repr))
	switch vm.State {
	case vbox.Poweroff:
		d.Set("status", "poweroff")
	case vbox.Running:
		d.Set("status", "running")
	case vbox.Paused:
		d.Set("status", "paused")
	case vbox.Saved:
		d.Set("status", "saved")
	case vbox.Aborted:
		d.Set("status", "aborted")
	}

	err = net_vbox_to_tf(vm, d)
	if err != nil {
		return err
	}

	/* Set connection info to first non NAT IPv4 address */
	for i, nic := range vm.NICs {
		if nic.Network == vbox.NICNetNAT {
			continue
		}
		availKey := fmt.Sprintf("network_adapter.%d.ipv4_address_available", i)
		if d.Get(availKey).(string) != "yes" {
			continue
		}
		ipv4Key := fmt.Sprintf("network_adapter.%d.ipv4_address", i)
		ipv4 := d.Get(ipv4Key).(string)
		if ipv4 == "" {
			continue
		}
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": ipv4,
		})
		break
	}

	return nil
}

func resourceVMUpdate(d *schema.ResourceData, meta interface{}) error {
	/* TODO: allow partial updates */

	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		return err
	}

	/* Stop VM */
	err = vm.Stop()
	if err != nil {
		return err
	}
	_, err = WaitForVMAttribute(d, "poweroff", []string{"running", "paused"}, "status", meta)
	if err != nil {
		return fmt.Errorf(
			"Error waiting for VM (%s) to become poweroff: %s", d.Get("name"), err)
	}

	/* Modify VM */
	err = tf_to_vbox(d, vm)
	if err != nil {
		return err
	}
	err = vm.Modify()
	if err != nil {
		return err
	}

	/* Start VM */
	err = vm.Start()
	if err != nil {
		return err
	}
	err = WaitUntilVMIsReady(d, vm, meta)
	if err != nil {
		return err
	}

	return resourceVMRead(d, meta)
}

func resourceVMDelete(d *schema.ResourceData, meta interface{}) error {
	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		return err
	}
	return vm.Delete()
}

/* Wait until VM is ready, and 'ready' means the first non NAT NIC get a ipv4_address assigned */
func WaitUntilVMIsReady(d *schema.ResourceData, vm *vbox.Machine, meta interface{}) error {
	var err error
	for i, nic := range vm.NICs {
		if nic.Network == vbox.NICNetNAT {
			continue
		}
		key := fmt.Sprintf("network_adapter.%d.ipv4_address_available", i)
		_, err = WaitForVMAttribute(d, "yes", []string{"", "no"}, key, meta)
		if err != nil {
			return fmt.Errorf(
				"Error waiting for VM (%s) to become ready: %s", d.Get("name"), err)
		}
		break
	}
	return err
}

func tf_to_vbox(d *schema.ResourceData, vm *vbox.Machine) error {
	var err error
	vm.OSType = "Linux_64"
	vm.CPUs = uint(d.Get("cpus").(int))
	bytes, err := humanize.ParseBytes(d.Get("memory").(string))
	vm.Memory = uint(bytes / humanize.MiByte) // VirtualBox expect memory to be in MiB units
	if err != nil {
		return err
	}
	vm.VRAM = 20 // Always 10MiB for vram
	vm.Flag = vbox.F_acpi | vbox.F_ioapic | vbox.F_rtcuseutc | vbox.F_pae |
		vbox.F_hwvirtex | vbox.F_nestedpaging | vbox.F_largepages | vbox.F_longmode |
		vbox.F_vtxvpid | vbox.F_vtxux
	vm.BootOrder = []string{"disk", "none", "none", "none"}
	vm.NICs, err = net_tf_to_vbox(d)
	return err
}

func net_tf_to_vbox(d *schema.ResourceData) ([]vbox.NIC, error) {
	tf_to_vbox_network_type := func(attr string) (vbox.NICNetwork, error) {
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

	tf_to_vbox_net_device := func(attr string) (vbox.NICHardware, error) {
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

	var err error
	var errs []error
	nicCount := d.Get("network_adapter.#").(int)
	adapters := make([]vbox.NIC, 0, nicCount)

	for i := 0; i < nicCount; i++ {
		prefix := fmt.Sprintf("network_adapter.%d.", i)
		var adapter vbox.NIC

		if attr, ok := d.Get(prefix + "type").(string); ok && attr != "" {
			adapter.Network, err = tf_to_vbox_network_type(attr)
		}
		if attr, ok := d.Get(prefix + "device").(string); ok && attr != "" {
			adapter.Hardware, err = tf_to_vbox_net_device(attr)
		}
		/* 'Hostonly' and 'bridged' network need property 'host_interface' been set */
		if adapter.Network == vbox.NICNetHostonly || adapter.Network == vbox.NICNetBridged {
			var ok bool
			adapter.HostInterface, ok = d.Get(prefix + "host_interface").(string)
			if !ok || adapter.HostInterface == "" {
				err = fmt.Errorf("'host_interface' property not set for '#%d' network adapter", i)
			}
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}

		log.Printf("[DEBUG] Network adapter: %+v\n", adapter)
		adapters = append(adapters, adapter)
	}

	if len(errs) > 0 {
		return nil, &multierror.Error{Errors: errs}
	}

	return adapters, nil
}

func net_vbox_to_tf(vm *vbox.Machine, d *schema.ResourceData) error {
	vbox_to_tf_network_type := func(netType vbox.NICNetwork) string {
		switch netType {
		case vbox.NICNetBridged:
			return "bridged"
		case vbox.NICNetNAT:
			return "nat"
		case vbox.NICNetHostonly:
			return "hostonly"
		case vbox.NICNetInternal:
			return "internal"
		case vbox.NICNetGeneric:
			return "generic"
		default:
			return ""
		}
	}

	vbox_to_tf_vdevice := func(vdevice vbox.NICHardware) string {
		switch vdevice {
		case vbox.AMDPCNetPCIII:
			return "PCIII"
		case vbox.AMDPCNetFASTIII:
			return "FASTIII"
		case vbox.IntelPro1000MTDesktop:
			return "IntelPro1000MTDesktop"
		case vbox.IntelPro1000TServer:
			return "IntelPro1000TServer"
		case vbox.IntelPro1000MTServer:
			return "IntelPro1000MTServer"
		default:
			return ""
		}
	}

	/* NICs in guest OS (eth0, eth1, etc) does not neccessarily have save
	order as in VirtualBox (nic1, nic2, etc), so we use MAC address to setup a mapping */
	type OsNicData struct {
		ipv4Addr string
		status   string
	}
	osNicMap := make(map[string]OsNicData) // map from MAC address to data

	/* Collect NIC data from guest OS */
	var errs []error
	for i := 0; i < len(vm.NICs); i++ {
		var osNic OsNicData

		/* NIC MAC address */
		macAddr, err := vm.GetGuestProperty(fmt.Sprintf("/VirtualBox/GuestInfo/Net/%d/MAC", i))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		/* NIC status */
		osNic.status, err = vm.GetGuestProperty(fmt.Sprintf("/VirtualBox/GuestInfo/Net/%d/Status", i))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		osNic.status = strings.ToLower(osNic.status)

		/* NIC ipv4 address */
		osNic.ipv4Addr, err = vm.GetGuestProperty(fmt.Sprintf("/VirtualBox/GuestInfo/Net/%d/V4/IP", i))
		if err != nil {
			errs = append(errs, err)
			continue
		}

		osNicMap[macAddr] = osNic
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	/* Assign NIC property to vbox structure and Terraform */
	nics := make([]map[string]interface{}, 0, 1)
	for _, nic := range vm.NICs {
		out := make(map[string]interface{})

		osNic, ok := osNicMap[nic.MacAddr]
		if !ok {
			errs = append(errs, fmt.Errorf("Could not find MAC address '%s' in guest OS", nic.MacAddr))
			continue
		}
		out["type"] = vbox_to_tf_network_type(nic.Network)
		out["device"] = vbox_to_tf_vdevice(nic.Hardware)
		out["host_interface"] = nic.HostInterface
		out["mac_address"] = nic.MacAddr
		out["status"] = osNic.status
		out["ipv4_address"] = osNic.ipv4Addr
		if osNic.ipv4Addr == "" {
			out["ipv4_address_available"] = "no"
		} else {
			out["ipv4_address_available"] = "yes"
		}

		nics = append(nics, out)
	}
	d.Set("network_adapter", nics)

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

func WaitForVMAttribute(
	d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}) (interface{}, error) {
	// Wait for the droplet so we can get the networking attributes
	// that show up after a while
	log.Printf(
		"[INFO] Waiting for VM (%s) to have %s of %s",
		d.Get("name"), attribute, target)

	stateConf := &resource.StateChangeConf{
		Pending:        pending,
		Target:         target,
		Refresh:        newVMStateRefreshFunc(d, attribute, meta),
		Timeout:        5 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	return stateConf.WaitForState()
}

func newVMStateRefreshFunc(
	d *schema.ResourceData, attribute string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		err := resourceVMRead(d, meta)
		if err != nil {
			return nil, "", err
		}

		attr := d.Get(attribute)
		log.Printf("=============== Refresh state '%s' : '%s'\n", attribute, attr.(string))

		// See if we can access our attribute
		if attr := d.Get(attribute); attr != "" {
			// Retrieve the VM properties
			vm, err := vbox.GetMachine(d.Id())
			if err != nil {
				return nil, "", fmt.Errorf("Error retrieving VM: %s", err)
			}

			return &vm, attr.(string), nil
		}

		return nil, "", nil
	}
}
