```mermaid
flowchart
  initrd -- exec --> mount_tmpfs
  mount_tmpfs -- exec --> populate
  populate -- switch --> sysroot
```