package virtualbox

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	humanize "github.com/dustin/go-humanize"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/pkg/errors"
	vbox "github.com/terra-farm/go-virtualbox"
)

var (
	vbm              string // Path to VBoxManage utility.
	defaultBootOrder = []string{"disk", "none", "none", "none"}
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

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"url": {
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Use the \"image\" option with a URL",
			},

			"optical_disks": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of Optical Disks to attach",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},

			"cpus": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  "2",
			},

			"memory": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "512mib",
			},

			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "running",
			},

			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"checksum": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"checksum_type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			"network_adapter": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"type": {
							Type:     schema.TypeString,
							Required: true,
						},

						"device": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "IntelPro1000MTServer",
						},

						"host_interface": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"ipv4_address": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"ipv4_address_available": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"boot_order": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Boot order, max 4 slots, each in [none, floopy, dvd, disk, net]",
				Elem:        &schema.Schema{Type: schema.TypeString},
				MaxItems:    4,
			},
		},
	}
}

func resourceVMExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	name := d.Get("name").(string)

	switch _, err := vbox.GetMachine(name); err {
	case nil:
		return true, nil
	case vbox.ErrMachineNotExist:
		return false, nil
	default:
		return false, errLogf("Checking existance of VM '%s': %v", name, err)
	}
}

var imageOpMutex sync.Mutex

func resourceVMCreate(d *schema.ResourceData, meta interface{}) error {
	image := d.Get("image").(string)

	if addr, exists := d.GetOk("url"); exists {
		image = addr.(string)
	}

	u, err := url.Parse(image)
	if err != nil {
		return fmt.Errorf("[Error] Could not parse image URL: %v", err)
	}

	imagePath, err := fetchIfRemote(u)
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to fetch remote image: %v", err)
	}

	/* Get gold folder and machine folder */
	usr, err := user.Current()
	if err != nil {
		return errLogf("Get the current user: %v", err)
	}
	goldFolder := filepath.Join(usr.HomeDir, ".terraform/virtualbox/gold")
	machineFolder := filepath.Join(usr.HomeDir, ".terraform/virtualbox/machine")
	os.MkdirAll(goldFolder, 0740)
	os.MkdirAll(machineFolder, 0740)

	// Unpack gold image to gold folder
	imageOpMutex.Lock() // Sequentialize image unpacking to avoid conflicts
	goldFileName := filepath.Base(imagePath)
	goldName := strings.TrimSuffix(goldFileName, filepath.Ext(goldFileName))
	if filepath.Ext(goldName) == ".tar" {
		goldName = strings.TrimSuffix(goldName, ".tar")
	}

	goldPath := filepath.Join(goldFolder, goldName)
	if err = unpackImage(imagePath, goldPath); err != nil {
		log.Printf("[ERROR] Unpack image %s: %s", imagePath, err.Error())
		imageOpMutex.Unlock()
		return errLogf("Unpacking image %s: %v", image, err)
	}
	imageOpMutex.Unlock()

	// Gather '*.vdi' and "*.vmdk" files from gold
	goldDisks, err := gatherDisks(goldPath)
	if err != nil {
		return errLogf("Unable to gather disks: %v", err)
	}

	// Create VM instance
	name := d.Get("name").(string)
	vm, err := vbox.CreateMachine(name, machineFolder)
	if err != nil {
		return errLogf("Create virtualbox VM %s: %v\n", name, err)
	}

	// Clone gold virtual disk files to VM folder
	for _, src := range goldDisks {
		filename := filepath.Base(src)

		target := filepath.Join(vm.BaseFolder, filename)
		vbm = "VBoxManage"
		if p := os.Getenv("VBOX_INSTALL_PATH"); p != "" && runtime.GOOS == "windows" {
			vbm = filepath.Join(p, "VBoxManage.exe")
		}
		setUUIDCmd := exec.Command(vbm, "internalcommands", "sethduuid", src)
		if err := setUUIDCmd.Run(); err != nil {
			return errLogf("Unable to set UUID: %v", err)
		}

		imageOpMutex.Lock() // Sequentialize image cloning to improve disk performance
		err := vbox.CloneHD(src, target)
		imageOpMutex.Unlock()
		if err != nil {
			return errLogf("Clone *.vdi and *.vmdk to VM folder: %v", err)
		}
	}

	// Attach virtual disks to VM
	vmDisks, err := gatherDisks(vm.BaseFolder)
	if err != nil {
		return errLogf("Unable to gather disks: %v", err)
	}

	if err := vm.AddStorageCtl("SATA", vbox.StorageController{
		SysBus:      vbox.SysBusSATA,
		Ports:       uint(len(vmDisks)) + 1,
		Chipset:     vbox.CtrlIntelAHCI,
		HostIOCache: true,
		Bootable:    true,
	}); err != nil {
		return errLogf("Create VirtualBox storage controller: %v", err)
	}

	for i, disk := range vmDisks {
		if err := vm.AttachStorage("SATA", vbox.StorageMedium{
			Port:      uint(i),
			Device:    0,
			DriveType: vbox.DriveHDD,
			Medium:    disk,
		}); err != nil {
			return errLogf("Attaching VirtualBox storage medium: %v", err)
		}
	}

	opticalDiskCount := d.Get("optical_disks.#").(int)
	opticalDisks := make([]string, 0, opticalDiskCount)

	for i := 0; i < opticalDiskCount; i++ {
		attr := fmt.Sprintf("optical_disks.%d", i)
		if opticalDiskImage, ok := d.Get(attr).(string); ok && attr != "" {
			opticalDisks = append(opticalDisks, opticalDiskImage)
		}
	}

	for i := 0; i < len(opticalDisks); i++ {
		opticalDiskImage := opticalDisks[i]
		fileName := filepath.Base(opticalDiskImage)

		target := filepath.Join(vm.BaseFolder, fileName)

		copyfile := exec.Command("cp", "-a", opticalDiskImage, target)
		if err := copyfile.Run(); err != nil {
			return errLogf("Cloning *.iso and *.dmg to VM folder: %v", err)
		}

		if err := vm.AttachStorage("SATA", vbox.StorageMedium{
			Port:      uint(len(vmDisks) + i),
			Device:    0,
			DriveType: vbox.DriveDVD,
			Medium:    target,
		}); err != nil {
			return errLogf("Attaching VirtualBox storage medium: %v", err)
		}
	}

	// Setup VM general properties
	if err := tfToVbox(d, vm); err != nil {
		return errLogf("Converting Terraform data to VM properties: %v", err)
	}
	if err := vm.Modify(); err != nil {
		return errLogf("Setup VM properties: %v", err)
	}

	// Start the VM
	if err := vm.Start(); err != nil {
		return errLogf("Starting VM: %v", err)
	}

	// Assign VM ID
	log.Printf("[DEBUG] Resource ID: %s\n", vm.UUID)
	d.SetId(vm.UUID)

	if err := waitUntilVMIsReady(d, vm, meta); err != nil {
		return errLogf("Wait VM until ready: %v", err)
	}

	// Errors here are already logged.
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
	switch err {
	case nil:
		break
	case vbox.ErrMachineNotExist:
		// VM no longer exists.
		d.SetId("")
		return nil
	default:
		return errLogf("unable to get machine: %v", err)
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
		return errLogf("can't get user data: %v", err)
	}
	if userData != nil && *userData != "" {
		d.Set("user_data", *userData)
	}

	if err = netVboxToTf(vm, d); err != nil {
		return errLogf("can't convert vbox network to terraform data: %v", err)
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

	d.Set("boot_order", vm.BootOrder)

	return nil
}

func powerOnAndWait(d *schema.ResourceData, vm *vbox.Machine, meta interface{}) error {
	if err := vm.Start(); err != nil {
		return errors.Wrap(err, "can't start vm")
	}

	return errors.Wrap(waitUntilVMIsReady(d, vm, meta), "unable to power on and wait")
}

func resourceVMUpdate(d *schema.ResourceData, meta interface{}) error {
	// TODO: allow partial updates

	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		return errLogf("unable to get machine: %v", d.Id(), err)
	}

	if err := vm.Poweroff(); err != nil {
		return errLogf("unable to poweroff machine: %v", d.Id(), err)
	}

	// Modify VM
	if err := tfToVbox(d, vm); err != nil {
		return errLogf("can't convert terraform config to virtual machine: %v", err)
	}
	if err := vm.Modify(); err != nil {
		return errLogf("unable to modify the vm: %v", err)
	}

	if err := powerOnAndWait(d, vm, meta); err != nil {
		return errLogf("unable to power on and wait for VM: %v", err)
	}

	// Errors are already logged
	return resourceVMRead(d, meta)
}

func resourceVMDelete(d *schema.ResourceData, meta interface{}) error {
	vm, err := vbox.GetMachine(d.Id())
	if err != nil {
		return errLogf("unable to get machine: %v", err)
	}
	if err := vm.Delete(); err != nil {
		return errLogf("unable to remove the VM: %v", err)
	}
	return nil
}

// Wait until VM is ready, and 'ready' means the first non NAT NIC get a ipv4_address assigned
func waitUntilVMIsReady(d *schema.ResourceData, vm *vbox.Machine, meta interface{}) error {
	for i, nic := range vm.NICs {
		if nic.Network == vbox.NICNetNAT {
			continue
		}

		key := fmt.Sprintf("network_adapter.%d.ipv4_address_available", i)
		if _, err := waitForVMAttribute(
			d, []string{"yes"}, []string{"no"}, key, meta, 3*time.Second, 3*time.Second,
		); err != nil {
			return errors.Wrapf(err, "waiting for VM (%s) to become ready", d.Get("name"))
		}
		break
	}
	return nil
}

func tfToVbox(d *schema.ResourceData, vm *vbox.Machine) error {
	var err error

	vm.OSType = "Linux_64"
	vm.CPUs = uint(d.Get("cpus").(int))
	bytes, err := humanize.ParseBytes(d.Get("memory").(string))
	if err != nil {
		return errors.Wrap(err, "cannot humanize bytes")
	}
	vm.Memory = uint(bytes / humanize.MiByte) // VirtualBox expect memory to be in MiB units

	vm.VRAM = 20 // Always 10MiB for vram
	vm.Flag = vbox.F_acpi | vbox.F_ioapic | vbox.F_rtcuseutc | vbox.F_pae |
		vbox.F_hwvirtex | vbox.F_nestedpaging | vbox.F_largepages | vbox.F_longmode |
		vbox.F_vtxvpid | vbox.F_vtxux
	vm.NICs, err = netTfToVbox(d)
	userData := d.Get("user_data").(string)
	if userData != "" {
		err = vm.SetExtraData("user_data", userData)
	}
	vm.BootOrder = defaultBootOrder
	for i, bootDev := range d.Get("boot_order").([]interface{}) {
		vm.BootOrder[i] = bootDev.(string)
	}
	return err
}

func netTfToVbox(d *schema.ResourceData) ([]vbox.NIC, error) {
	tfToVboxNetworkType := func(attr string) (vbox.NICNetwork, error) {
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
			return "", fmt.Errorf("Invalid virtual network adapter type: %s", attr)
		}
	}

	tfToVboxNetDevice := func(attr string) (vbox.NICHardware, error) {
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
			return "", fmt.Errorf("Invalid virtual network device: %s", attr)
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
			adapter.Network, err = tfToVboxNetworkType(attr)
		}
		if attr, ok := d.Get(prefix + "device").(string); ok && attr != "" {
			adapter.Hardware, err = tfToVboxNetDevice(attr)
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

// countRuntimeNics will return the number of NICs found after VM successfully started.
func countRuntimeNICs(vm *vbox.Machine) (int, error) {
	count, err := vm.GetGuestProperty("/VirtualBox/GuestInfo/Net/Count")

	if err != nil {
		return 0, err
	}

	if count == nil {
		return 0, nil
	}

	return strconv.Atoi(*count)
}

func netVboxToTf(vm *vbox.Machine, d *schema.ResourceData) error {
	vboxToTfNetworkType := func(netType vbox.NICNetwork) string {
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

	vboxToTfVdevice := func(vdevice vbox.NICHardware) string {
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
		nicCount, err := countRuntimeNICs(vm)
		if err != nil {
			return err
		}

		if nicCount < len(vm.NICs) {
			return nil
		}

		/* NICs in guest OS (eth0, eth1, etc) does not neccessarily have save
		order as in VirtualBox (nic1, nic2, etc), so we use MAC address to setup a mapping */
		type OsNicData struct {
			ipv4Addr string
			status   string
		}
		osNicMap := make(map[string]OsNicData) // map from MAC address to data

		var errs []error
		for i := 0; i < nicCount; i++ {
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

		// Assign NIC property to vbox structure and Terraform
		nics := make([]map[string]interface{}, 0, 1)

		for _, nic := range vm.NICs {
			out := make(map[string]interface{})

			out["type"] = vboxToTfNetworkType(nic.Network)
			out["device"] = vboxToTfVdevice(nic.Hardware)
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
		// Assign NIC property to vbox structure and Terraform
		nics := make([]map[string]interface{}, 0, 1)

		for _, nic := range vm.NICs {
			out := make(map[string]interface{})

			out["type"] = vboxToTfNetworkType(nic.Network)
			out["device"] = vboxToTfVdevice(nic.Hardware)
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
func waitForVMAttribute(
	d *schema.ResourceData, target []string, pending []string, attribute string, meta interface{}, delay, interval time.Duration) (interface{}, error) {
	// Wait for the vm so we can get the networking attributes that show up
	// after a while.
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
				return nil, "", errors.Wrap(err, "unable to retrive vm")
			}

			return &vm, attr.(string), nil
		}

		return nil, "", nil
	}
}

func fetchIfRemote(u *url.URL) (string, error) {
	// If the schema is empty, treat it as a local path, otherwise
	// use it as a remote.
	if u.Scheme == "" {
		return u.Path, nil
	}

	// TODO: Add special handing for other schemes, such as
	// 		 s3, gcs, (s)ftp(s).
	// We want to quit if the scheme is not currently supported.
	switch u.Scheme {
	case "http", "https":
		break
	default:
		return "", fmt.Errorf("unsupported scheme %s", u.Scheme)
	}

	_, file := filepath.Split(u.Path)

	// if the file is not found, and the error is unexpected, return
	if _, err := os.Stat(file); err != nil && !os.IsNotExist(err) {
		return "", err
	}

	f, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}

	return file, nil
}
