# simplek8s.yaml

## Configuration specifications

The file `simplek8s.yaml` is a YAML document conforming to the following specification:

> Italicized entries being optional.

- `version` (string): Currently must be `1`
- _`users`_ (list of objects):
  - `name` (string):
  - _`passwordHash`_ (string): Default is "x".
  - _`sshAuthorizedKeys`_ (list of string): Default is none.
- _`storage`_ (object):
  - _`mounts`_ (list of objects):
    - `what` (string)
    - `where` (string)
    - _`type`_ (string): Default is "auto".
    - _`options`_ (string): Default is "defaults".
  - _`links`_ (list of objects):
    - _`overwrite`_ (bool): Default is false.
    - `path` (string)
    - `target` (string)
    - _`owner`_ (string): Default is the current process uid:gid.
    - _`hard`_ (bool): Default is false.
  - _`directories`_ (list of objects):
    - _`overwrite`_ (bool): Default is false.
    - `path` (string)
    - _`owner`_ (string): Default is the current process uid:gid.
    - _`permissions`_ (string): Default is "0775".
  - _`files`_ (list of objects):
    - _`overwrite`_ (bool): Default is false.
    - `path` (string)
    - _`encoding`_ (string): If it is "b64", the field `content` will be decoded as base64. Default is empty.
    - _`content`_ (string): Default is empty.
    - _`owner`_ (string): Default is the current running process uid:gid.
    - _`permissions`_ (string): Default is "0664".


### Minimal example

```yaml
version: "1"
users:
  - name: "root"
    sshAuthorizedKeys:
      - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICOJHHXFdcBLMAviMAHgQvCuzpnLmzXxatIL6IUe7b6W salvador.joseluis@gmail.com"
storage:
  mounts:
    - what: /dev/sda2
      where: /var
```


### Full example below

```yaml
version: "1"

users:
  - name: "root"
    # Password will be `root`
    passwordHash: "\$6\$n2yzXErLUUm5/C38\$PtFZeevgw7A5LlD7J8WZlElsIl6yfpse4F6MeVx0GhIHn9WdHkVePiyd2x/kQE2UeJit.tKPn/Yez0fvL9O0K."
    sshAuthorizedKeys:
      - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICOJHHXFdcBLMAviMAHgQvCuzpnLmzXxatIL6IUe7b6W salvador.joseluis@gmail.com"

storage:
  mounts:
    - what: "/dev/sda2"
      where: "/var"
      type: "ext4"
      options: "defaults"

    - what: "/var/home"
      where: "/home"
      type: "none"
      options: "bind"

    - what: "10.0.0.123:/storage"
      where: "/mnt/storage"
      type: "nfs4"
      options: "defaults"

  links:
    - overwrite: false
      path: /etc/somedir
      target: /root/somedir
      owner: root:root
      hard: false

    - overwrite: true
      # Mask `/usr/lib/systemd/network/99-dhcp.network`
      path: /etc/systemd/network/99-dhcp.network
      target: /dev/null
      owner: root:root
      hard: false

  directories:
    - overwrite: false
      path: /root/somedir
      owner: root:root
      permissions: "0770"

    - overwrite: true # This flag will empty the `/root/will_empty_dir` directory
      path: /root/will_empty_dir
      owner: root:root
      permissions: "0775"

  files:
    - # A example file
      overwrite: false
      path: /root/somedir/somefile
      encoding: b64
      # `content` will be decoded as: Hello world
      content: SGVsbG8gd29ybGQK
      owner: root:root
      permissions: "0664"

    - # Add a global executable script
      overwrite: true
      path: /usr/local/bin/hello
      encoding:
      content: |
        #!/bin/sh
        echo "Hello world"
      owner: root:root
      permissions: "0775"

    - # Configure static network address
      overwrite: false
      path: /etc/systemd/network/50-wired.network
      encoding:
      content: |
        [Match]
        Name=en*

        [Network]
        Address=192.168.1.50/24
        Gateway=192.168.1.1
        DNS=192.168.1.1
        DNS=1.1.1.1
        DNS=8.8.8.8
      owner: root:root
      permissions: "0644"
```

## Roadmap

- Fetch external file content from http or https.
- Validate external file content by GPG.
