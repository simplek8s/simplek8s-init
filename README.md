## simplek8s.yaml

### Configuration specifications

The file `simplek8s.yaml` is a YAML document conforming to the following specification:

> Italicized entries being optional.

- `version` (string): Currently must be `1.0.0`
- _`users`_ (list of objects):
  - `name` (string):
  - `passwordHash` (string):
  - `sshAuthorizedKeys` (string):
- _`storage`_ (object):
  - _`mounts`_ (list of objects):
  - _`links`_ (list of objects):
  - _`directories`_ (list of objects):
  - _`files`_ (list of objects):

Full example below:

```yaml
version: "1.0.0"
users:
  - name: "root"
    passwordHash: "\$6\$n2yzXErLUUm5/C38\$PtFZeevgw7A5LlD7J8WZlElsIl6yfpse4F6MeVx0GhIHn9WdHkVePiyd2x/kQE2UeJit.tKPn/Yez0fvL9O0K."
    sshAuthorizedKeys:
      - "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICOJHHXFdcBLMAviMAHgQvCuzpnLmzXxatIL6IUe7b6W salvador.joseluis@gmail.com"
storage:
  - mounts:
      - what: /dev/sda2
        where: /var
        type: auto
        options: defaults
  - links:
      - overwrite: false
        path: /etc/somedir
        target: /root/somedir
        owner: root:root
        hard: false
  - directories:
      - overwrite: false
        path: /root/somedir
        owner: root:root
        permissions: "0750"
      - overwrite: true # This flag will empty the `/root/will_empty_dir` directory
        path: /root/will_empty_dir
        owner: root:root
        permissions: "0750"
  - files:
      - overwrite: false
        path: /root/somedir/somefile
        encoding: b64
        content: SGVsbG8gd29ybGQK
        owner: root:root
        permissions: "0644"
      - overwrite: true
        path: /usr/local/bin/hello
        encoding:
        content: |
          #!/bin/sh
          echo "Hello world"
        owner: root:root
        permissions: "0755"
```

### Roadmap

- Fetch external file content from http or https.
- Validate external file content by GPG.
