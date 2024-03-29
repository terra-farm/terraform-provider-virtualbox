---
layout: "virtualbox"
page_title: "Provider: Virtualbox"
sidebar_current: "docs-virtualbox-index"
description: |-
  Virtualbox provider for Terraform.
---

# Virtualbox Provider

The Virtualbox provider for Terraform allows managing local VirtualBox machines
using Terraform. The main purpose of this provider is to make you familiar with
Terraform and provisioning machines, without leaving your machine, therefore
saving you costs. However, remember that your local environment might differ
from a cloud provider.

## Example Usage

```hcl
terraform {
  required_providers {
    virtualbox = {
      source = "terra-farm/virtualbox"
      version = "0.2.1"
    }
  }
}

# In general, you can use the provider without the need to set any configuration options.
# However, should you want to adjust how long it will wait for a VM to become ready,
# you can use the following block. Uncomment it and adjust the values (they are in seconds).
# provider "virtualbox" {
#   ready_delay   = 60
#   ready_timeout = 5
# }

resource "virtualbox_vm" "vm1" {
  name   = "debian-11"
  image  = "https://app.vagrantup.com/generic/boxes/debian11/versions/4.3.12/providers/virtualbox/amd64/vagrant.box"
  cpus      = 1
  memory    = "512 mib"
  user_data = file("${path.module}/user_data")

  network_adapter {
    type           = "hostonly"
    device         = "IntelPro1000MTDesktop"
    host_interface = "vboxnet1"
    # On Windows use this instead
    # host_interface = "VirtualBox Host-Only Ethernet Adapter"
  }
}

output "IPAddress" {
  value = element(virtualbox_vm.vm1.*.network_adapter.0.ipv4_address, 1)
}

```
