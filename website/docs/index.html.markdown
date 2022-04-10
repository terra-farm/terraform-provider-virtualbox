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
      source = "shekeriev/virtualbox"
      version = "0.0.4"
    }
  }
}

provider "virtualbox" {
  delay      = 60
  mintimeout = 5
}

resource "virtualbox_vm" "vm1" {
  name   = "debian-11"
  image  = "https://app.vagrantup.com/shekeriev/boxes/debian-11/versions/0.2/providers/virtualbox.box"
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
