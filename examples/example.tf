
resource "virtualbox_vm" "node" {
    count = 2
    name = "${format("node-%02d", count.index+1)}"
    url = "https://vagrantcloud.com/ubuntu/boxes/trusty64/versions/20180206.0.0/providers/virtualbox.box"
    image = "./virtualbox-ubuntu.box"
    cpus = 2
    memory = "512 mib",
    user_data = "${file("user_data")}"


    network_adapter {
        type = "bridged",
        host_interface = "en0",

    }


}
output "IPAddr" {
    value = "${element(virtualbox_vm.node.*.network_adapter.0.ipv4_address, 1)}"
}
output "IPAddr_2" {
    value = "${element(virtualbox_vm.node.*.network_adapter.0.ipv4_address, 2)}"
}
