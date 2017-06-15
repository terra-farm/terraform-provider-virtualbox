package provider

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/hashicorp/terraform/helper/resource"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	vbox "github.com/pyToshka/go-virtualbox"
	"github.com/hashicorp/terraform/helper/schema"
	multierror "github.com/hashicorp/go-multierror"
	"os/exec"
	"runtime"
	"io"
	"net/http"
)
var (
	VBM     string // Path to VBoxManage utility.
)

func init() {
	vbox.Verbose = true
}

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

			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
				Optional: true,
				Default:  "running",
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"checksum": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"checksum_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
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

func resourceVMExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	name := d.Get("name").(string)
	_, err := vbox.GetMachine(name)
	if err == nil {
		return true, nil
	} else if err == vbox.ErrMachineNotExist {
		return false, nil
	} else {
		log.Printf("[ERROR] Checking existence of VM '%s'\n", name)
		return false, err
	}
}

var imageOpMutex sync.Mutex

func resourceVMCreate(d *schema.ResourceData, meta interface{}) error {
	/* TODO: allow partial updates */
	var _, err = os.Stat(d.Get("image").(string))
	if os.IsNotExist(err) {
		if len(d.Get("url").(string)) > 0 {
			if len(d.Get("url").(string)) > 0 {
				path := d.Get("image").(string)
				url := d.Get("url").(string)
				// Create the file
				out, err := os.Create(path)
				if err != nil {
					return err
				}
				defer out.Close()
				// Get the data
				resp, err := http.Get(url)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				// Writer the body to file
				_, err = io.Copy(out, resp.Body)
				if err != nil {
					return err
				}
			}
		}
	}
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
	imageOpMutex.Lock() // Sequentialize image unpacking to avoid conflicts
	goldFileName := filepath.Base(image)
	goldName := strings.TrimSuffix(goldFileName, filepath.Ext(goldFileName))
	if filepath.Ext(goldName) == ".tar" {
		goldName = strings.TrimSuffix(goldName, ".tar")
	}
	goldPath := filepath.Join(goldFolder, goldName)
	err = unpackImage(image, goldPath)
	if err != nil {
		log.Printf("[ERROR] Unpack image %s: %s", image, err.Error())
		imageOpMutex.Unlock()
		return err
	}
	imageOpMutex.Unlock()

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
		VBM = "VBoxManage"
		if p := os.Getenv("VBOX_INSTALL_PATH"); p != "" && runtime.GOOS == "windows" {
			VBM = filepath.Join(p, "VBoxManage.exe")
		}
		setuiid := exec.Command(VBM + "internalcommands sethduuid " +src)
		err := setuiid.Run()
		imageOpMutex.Lock() // Sequentialize image cloning to improve disk performance
		err = vbox.CloneHD(src, target)
		imageOpMutex.Unlock()
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

func setState(d *schema.ResourceData, state vbox.MachineState) {
	switch state {
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
}

func resourceVMRead(d *schema.ResourceData, meta interface{}) error {
	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		/* VM no longer exist */
		if err == vbox.ErrMachineNotExist {
			d.SetId("")
			return nil
		}
		return err
	}

	// if vm.State != vbox.Running {
	// 	setState(d, vm.State)
	// 	return nil
	// }

	setState(d, vm.State)
	d.Set("name", vm.Name)
	d.Set("cpus", vm.CPUs)
	bytes := uint64(vm.Memory) * humanize.MiByte
	repr := humanize.IBytes(bytes)
	d.Set("memory", strings.ToLower(repr))

	userData, err := vm.GetExtraData("user_data")
	if err != nil {
		return err
	}
	if userData != nil && *userData != "" {
		d.Set("user_data", *userData)
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

func powerOnAndWait(d *schema.ResourceData, vm *vbox.Machine, meta interface{}) error {
	if err := vm.Start(); err != nil {
		return err
	}

	return WaitUntilVMIsReady(d, vm, meta)
}

func resourceVMUpdate(d *schema.ResourceData, meta interface{}) error {
	/* TODO: allow partial updates */

	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		return err
	}

	if err := vm.Poweroff(); err != nil {
		return err
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

	if err := powerOnAndWait(d, vm, meta); err != nil {
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
		_, err = WaitForVMAttribute(d,[]string{"yes"}, []string{"no"}, key, meta, 3*time.Second, 3*time.Second)
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
	userData := d.Get("user_data").(string)
	if userData != "" {
		err = vm.SetExtraData("user_data", userData)
	}
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

	/* Collect NIC data from guest OS, available only when VM is running */
	if vm.State == vbox.Running {
		/* NICs in guest OS (eth0, eth1, etc) does not neccessarily have save
		order as in VirtualBox (nic1, nic2, etc), so we use MAC address to setup a mapping */
		type OsNicData struct {
			ipv4Addr string
			status   string
		}
		osNicMap := make(map[string]OsNicData) // map from MAC address to data

		var errs []error

		for i := 0; i < len(vm.NICs); i++ {
			var osNic OsNicData

			/* NIC MAC address */
			macAddr, err := vm.GetGuestProperty(fmt.Sprintf("/VirtualBox/GuestInfo/Net/%d/MAC", i))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if macAddr == nil || *macAddr == "" {
				return nil
			}

			/* NIC status */
			status, err := vm.GetGuestProperty(fmt.Sprintf("/VirtualBox/GuestInfo/Net/%d/Status", i))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if status == nil || *status == "" {
				return nil
			}
			osNic.status = strings.ToLower(*status)

			/* NIC ipv4 address */
			ipv4Addr, err := vm.GetGuestProperty(fmt.Sprintf("/VirtualBox/GuestInfo/Net/%d/V4/IP", i))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if ipv4Addr == nil || *ipv4Addr == "" {
				return nil
			}
			osNic.ipv4Addr = *ipv4Addr

			osNicMap[*macAddr] = osNic
		}

		if len(errs) > 0 {
			return &multierror.Error{Errors: errs}
		}

		/* Assign NIC property to vbox structure and Terraform */
		nics := make([]map[string]interface{}, 0, 1)

		for _, nic := range vm.NICs {
			out := make(map[string]interface{})

			out["type"] = vbox_to_tf_network_type(nic.Network)
			out["device"] = vbox_to_tf_vdevice(nic.Hardware)
			out["host_interface"] = nic.HostInterface
			out["mac_address"] = nic.MacAddr

			osNic, ok := osNicMap[nic.MacAddr]
			if !ok {
				return nil
			}
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
	} else {
		/* Assign NIC property to vbox structure and Terraform */
		nics := make([]map[string]interface{}, 0, 1)

		for _, nic := range vm.NICs {
			out := make(map[string]interface{})

			out["type"] = vbox_to_tf_network_type(nic.Network)
			out["device"] = vbox_to_tf_vdevice(nic.Hardware)
			out["host_interface"] = nic.HostInterface
			out["mac_address"] = nic.MacAddr

			out["status"] = "down"
			out["ipv4_address"] = ""
			out["ipv4_address_available"] = "no"

			nics = append(nics, out)
		}

		d.Set("network_adapter", nics)
	}

	return nil
}

//func WaitForVMAttribute(d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}, delay, interval time.Duration) (interface{}, error) {
//	// Wait for the droplet so we can get the networking attributes
//	// that show up after a while
//	log.Printf(
//		"[INFO] Waiting for VM (%s) to have %s of %s",
//		d.Get("name"), attribute, target)
//
//	stateConf := &resource.StateChangeConf{
//		Pending:        pending,
//		Target:         []string{target},
//		Refresh:        newVMStateRefreshFunc(d, attribute, meta),
//		Timeout:        5 * time.Minute,
//		Delay:          delay,
//		MinTimeout:     interval,
//		NotFoundChecks: 60,
//	}
//
//	return stateConf.WaitForState()
//}
func WaitForVMAttribute(
	d *schema.ResourceData, target []string, pending []string, attribute string, meta interface{}, delay, interval time.Duration) (interface{}, error) {
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
		Delay:          delay,
		MinTimeout:     interval,
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

		// See if we can access our attribute
		if attr, ok := d.GetOk(attribute); ok {
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
