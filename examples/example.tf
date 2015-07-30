
resource "virtualbox_vm" "node01" {
    name = "node01"

    image = "/Users/cailei/ubuntu-15.04.tar.xz"
    cpus = 2
    memory = "512mib"

    network_adapter {
        type = "nat"
    }

    network_adapter {
        type = "bridged"
        host_interface = "en0"
        # "en0: 以太网"
    }
}

output "IP address" {
    value = "${virtualbox_vm.node01.network_adapter.1.ipv4_address}"
}

