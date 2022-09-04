---
name: Bug report
about: Report a Bug
title: "[BUG] <Title>"
labels: bug
assignees: VoyTechnology

---

# WAIT

Before you submit a new bug, make sure to read through existing open bugs. If you are unsure if something is a bug or if you would like to ask more questions, use Github Discussions. Please remove this paragraph before submitting the bug as a confirmation that you have read this message and to the best of your ability you looked at the other issues and do not see another issue like the one you are experiencing.

## Terraform Version

`eg. v1.2.8`

## Virtualbox Provider version

`eg. v0.2.2-alpha.2`

## Operating System

`eg. Linux`

## Describe the bug

A clear and concise description of what the bug is.

## To Reproduce

Steps to reproduce the behavior:
1. Step 1
2. Step 2
3. Step 3

## Expected Behaviour

A clear and concise description of what you expected to happen.

## Configuration

```tf
resource "virtualbox_vm" "vm" {
  // …
}
```

## Log Output

```
Line by line log
or
pastebin link or other service
```

If you feel like the log does not accurately represent the error, try increasing the log verbosity: https://www.terraform.io/plugin/log/managing
