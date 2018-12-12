# v.0.1.2
- Add support for using external vagrant box and archives as source for deploy

# v.0.1.1.1

- Add support for Terraform v0.9.5

# v0.1.1

- Add new optional field "user_data" in schema, accepts arbitary string, your VM specific configuration can be stored here.
  Some 3rdparty tool expects certain content be set in this field.
  For example, [mantl/terraform.py](https://github.com/mantl/terraform.py) (a terraform-ansible bridging tool) interprete 'user_data' as a JSON string:
  ```
  {
    "role": "foobar",
    ...
  }
  ```
  It uses the 'role' field to group hosts.
