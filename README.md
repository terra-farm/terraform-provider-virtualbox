[![Build Status](https://github.com/terra-farm/terraform-provider-virtualbox/workflows/CI/badge.svg)](https://github.com/terra-farm/terraform-provider-virtualbox/actions?query=branch%3Amaster)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fterra-farm%2Fterraform-provider-virtualbox.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fterra-farm%2Fterraform-provider-virtualbox?ref=badge_shield)
[![Gitter](https://badges.gitter.im/terra-farm/terraform-provider-virtualbox.svg)](https://gitter.im/terra-farm/terraform-provider-virtualbox?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

# VirtualBox provider for Terraform

Published documentation is located on the [Terraform Registry](https://registry.terraform.io/providers/terra-farm/virtualbox/latest/docs)

## Maintainers Needed

[__We are looking for additional maintainers.__](https://github.com/terra-farm/terraform-provider-virtualbox/discussions/117)

## Usage

```tf
terraform {
  required_providers {
    virtualbox = {
      source = "terra-farm/virtualbox"
      version = "<latest-tag>"
    }
  }
}

provider "virtualbox" {
  # Configuration options
}

resource "virtualbox_vm" "vm" {
  // ...
}
```

## Example

You can find a practical example in the [`/examples` directory](/examples)

If you want to contribute documentation changes, see the [Contribution guide](CONTRIBUTING.md).

## Limitations

- __Experimental provider!__
- We only officially support the latest version of Go, Virtualbox and Terraform. The provider might be compatible and work with other versions
  but we do not provide any level of support for this due to lack of time.
- The defaults here are only tested with the [vagrant insecure (packer) keys](https://github.com/hashicorp/vagrant/tree/master/keys) as the login.

## Contributors

Special thanks to all contributors, and [@ccll](https://github.com/ccll) for donating the original project to the terra-farm group!

Inspired by [terraform-provider-vix](https://github.com/hooklift/terraform-provider-vix)

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fterra-farm%2Fterraform-provider-virtualbox.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fterra-farm%2Fterraform-provider-virtualbox?ref=badge_large)
