# SimpleK8s Init

![SimpleK8s Init Logo](assets/icon.png)

**SimpleK8s Init** configures and populates the **initrd** and **sysroot** boot
stages during early system initialization.

During the initrd stage, SimpleK8s Init searches all FAT32 partitions for a
`simplek8s.yaml` configuration file, which defines how the real root filesystem
should be populated.

The `simplek8s.yaml` file specifies all users, groups, files, mounts, and other
resources to be created during these stages.

Once the sysroot has been prepared, SimpleK8s Init hands control over to
**systemd**, which continues booting into the real system.

## 📄 Documentation

| Topic                                    | Description                                                      |
| ---------------------------------------- | ---------------------------------------------------------------- |
| [Boot Sequence](docs/boot-sequence.md)   | A detailed explanation of the SimpleK8s Init boot sequence.      |
| [simplek8s.yaml](docs/simplek8s-yaml.md) | How to configure `simplek8s.yaml`, including practical examples. |

## 🤝 Contributing

Contributions are welcome! Feel free to open issues, submit pull requests, or
suggest new features to help improve SimpleK8s Init.

## ©️ License

This project is licensed under the [Apache 2.0 License](LICENSE).
