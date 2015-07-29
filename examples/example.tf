
provider "vix" {
    // valid options are: "fusion", "workstation", "serverv1", "serverv2", "player"
    // and "workstation_shared"
    product = "fusion"
    verify_ssl = false

    // clone_type can be "full" or "linked". Advantages of one over the other
    // are described here: https://www.vmware.com/support/ws5/doc/ws_clone_typeofclone.html
    // cloning_strategy = "linked"
}

/*resource "vix_vswitch" "vmnet10" {
    name = "vmnet10"
    nat = true
    dhcp = true
    range = "192.168.1.0/24"
    host_access = true
}*/


resource "vix_vm" "core01" {
    name = "core01"
    description = "Terraform VMWARE VIX test"

    /*
    * The provider will download, verify, decompress and untar the image.
    * Ideally you will provide images that have VMware Tools installed already,
    * otherwise the provider will be considerably limited for what it can do.
    */
    image {
        url = "https://github.com/c4milo/dobby-boxes/releases/download/stable/coreos-stable-vmware.box"
        checksum = "5b20ba225b5d4e1a3c01e071f8c9f607ed6c6e1d88e6ed5a66387aaf75b9ee2b"
        checksum_type = "sha256"

        // If image is encrypted we need to provide a password
        // password = "${var.password}"
    }

    cpus = 2

    // Memory sizes must be provided using IEC sizes such as: kib, ki, mib, mi,
    // gib or gi.
    memory = "1.0gib"
    upgrade_vhardware = false
    tools_init_timeout = "15s"

    // Be aware that GUI does not work if the virtual machine is encrypted
    gui = true

    // Whether to enable or disable all shared folders for this VM
    sharedfolders = true

    // Advanced configuration
    /*network_adapter {
        // type can be either "custom", "nat", "bridged" or "hostonly"
        type = "custom"
        mac_address = "00:00:00:00:00"
        mac_address_type = "static"

        // vswitch is only required when network type is "custom"
        vswitch = ${vix_vswitch.vmnet10.name}

        // vmxnet3 requires VMware Tools
        driver = "vmxnet3"
    }*/

    // Minimal required
    network_adapter {
        type = "bridged"
    }

    network_adapter {
        type = "nat"
        mac_address = "00:50:56:aa:bb:cc"
        mac_address_type = "static"
    }

    network_adapter {
        type = "hostonly"
    }

    cdrom {
        bus_type = "scsi"
        image = "/Users/camilo/Dropbox/Development/cloudescape/hooklift-boxes/cfgdrv.iso"
    }

    // Adds an IDE device by default and autodetects the host's CDROM
    cdrom {}

    shared_folder {
        name = "Dev1"
        enable = false
        guest_path = "/home/camilo/dev"
        host_path = "/Users/camilo/Development"
        readonly = false
    }
}

output "IP address" {
    value = "${vix_vm.core01.ip_address}"
}

