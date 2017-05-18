
resource "virtualbox_vm" "node" {
    count = 2
    name = "${format("node-%02d", count.index+1)}"

    image = "./ubuntu-15.04.tar.xz"
    cpus = 2
    memory = "512mib"

    user_data = "${file("user_data")}"

    network_adapter {
        type = "nat"
    }

    network_adapter {
        type = "bridged"
        host_interface = "en0"
    }
}

output "IPAddr" {
    value = "${element(virtualbox_vm.node.*.network_adapter.1.ipv4_address, 1)}"
}

