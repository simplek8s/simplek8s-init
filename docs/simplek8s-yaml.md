# simplek8s.yaml

The file `simplek8s.yaml` is a YAML document conforming to the following specification:

## Specifications

> All fields marked in italic are optional. Deprecated fields are struck through.

- `version` (string): Currently must be `1`.
- _`users`_ (list of objects):
  - `name` (string): User name.
  - _`password_hash`_ (string): Default is "x".
  - **Deprecated**: ~~_`passwordHash`_~~ (string): Default is "x".
  - _`ssh_authorized_keys`_ (list of string): Default is none.
  - **Deprecated**: ~~_`sshAuthorizedKeys`_~~ (list of string): Default is none.
  - _`uid`_ (int): Default is next unused uid.
  - _`gid`_ (int): Default is the same value of uid.
  - _`groups`_ (list of string): Default is none.
  - _`system`_ (boolean): UID default value will be more or equal than 1000 if
    it is false. Default is false.
  - _`gecos`_ (string): User description. Default is empty.
- _`groups`_ (list of objects):
  - `name` (string): Group name.
  - _`gid`_ (int): Group Id. Default is next unused gid.
  - _`system`_ (boolean): GID default value will be more or equal than 1000 if
    it is false. Default is false.
- _`storage`_ (object):
  - _`mounts`_ (list of objects):
    - `what` (string): Device or path.
    - `where` (string): Target mount point path.
    - _`type`_ (string): Filesystem type. Default is "auto".
    - _`options`_ (string): Default is "defaults".
    - **Deprecated**: ~~_`after`_~~ (list of string):
  - _`links`_ (list of objects):
    - _`overwrite`_ (boolean): Default is false.
    - `path` (string)
    - `target` (string)
    - _`owner`_ (string): Default is the current process uid:gid.
    - _`hard`_ (boolean): Default is false.
  - _`directories`_ (list of objects):
    - _`overwrite`_ (boolean): Default is false.
    - `path` (string)
    - _`owner`_ (string): Default is the current process uid:gid.
    - _`permissions`_ (string): Default is "0775".
  - _`files`_ (list of objects):
    - _`overwrite`_ (boolean): Default is false.
    - `path` (string)
    - _`encoding`_ (string): Default is empty.
      If it is "b64" `content` will be decoded as Base64.
    - _`content`_ (string): Default is empty.
    - _`owner`_ (string): Default is the current running process uid:gid.
    - _`permissions`_ (string): Default is "0664".

### Minimal example

```yaml
version: "1"

users:
  - name: root
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICOJHHXFdcBLMAviMAHgQvCuzpnLmzXxatIL6IUe7b6W salvador.joseluis@gmail.com

storage:
  mounts:
    - what: /dev/disk/by-label/var
      where: /var
```

### Full example below

```yaml
version: "1"

users:
  - name: root
    # Password will be `root`
    password_hash: \$6\$n2yzXErLUUm5/C38\$PtFZeevgw7A5LlD7J8WZlElsIl6yfpse4F6MeVx0GhIHn9WdHkVePiyd2x/kQE2UeJit.tKPn/Yez0fvL9O0K.
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICOJHHXFdcBLMAviMAHgQvCuzpnLmzXxatIL6IUe7b6W salvador.joseluis@gmail.com
    uid: 0
    gid: 0
    groups:
      - ops
    system: false
    gecos: Super User

groups:
  - name: ops
    gid: 1
    system: true

storage:
  mounts:
    - what: /dev/disk/by-label/var
      where: /var
      type: ext4
      options: defaults

    - what: /var/home
      where: /home
      type: none
      options: bind

    - what: 10.0.0.123:/storage
      where: /mnt/storage
      type: nfs4
      options: defaults

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
    - # An example file
      overwrite: false
      path: /root/somedir/somefile
      encoding: b64 # Base64-encoded content
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

- Support fetching external file content over HTTP/HTTPS.
- Add GPG signature verification for externally fetched content.
