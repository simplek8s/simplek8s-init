// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package blkid fetch device attributes.
package blkid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

var ErrUnknownFilesystem = errors.New("unknown filesystem type")

// DetectFileSystem detects the file system type of a given device.
//
// Currently, the next filesystems are supported:
//   - APFS
//   - Bcache
//   - BcacheFS
//   - BEFS
//   - BFS
//   - Btrfs
//   - CRAMFS
//   - EROFS
//   - ExFAT
//   - EXFS
//   - ext2/ext3/ext4
//   - F2FS
//   - FAT32
//   - GFS
//   - HFS
//   - HFS +Journal
//   - HPFS
//   - ISO9660
//   - JFFS2
//   - JFS
//   - Minix
//   - NILFS
//   - NTFS
//   - OCFS
//   - QNX4
//   - QNX6
//   - Reiser4
//   - ReiserFS
//   - ROMFS
//   - SquashFS
//   - SquashFS3
//   - SWAP
//   - UBIFS
//   - UDF
//   - UFS
//   - UFS2
//   - XFS
//   - XIAFS
//   - ZFS
func DetectFileSystem(device string) (string, error) {
	file, err := os.Open(device)
	if err != nil {
		return "", fmt.Errorf("cannot open device %s: %w", device, err)
	}
	defer file.Close()

	for _, detect := range []func(*os.File) string{
		detectAPFS,
		detectBcache,
		detectBcacheFS,
		detectBEFS,
		detectBFS,
		detectBtrfs,
		detectCramFS,
		detectEROFS,
		detectExFAT,
		detectEXFS,
		detectExt,
		detectF2FS,
		detectFAT,
		detectGFS,
		detectHFSPlusJournaled, // Must be before HFS.
		detectHFS,
		detectHPFS,
		detectUDF, // Must be before ISO9660.
		detectISO9660,
		detectJFFS2,
		detectJFS,
		detectMinix,
		detectNILFS,
		detectNTFS,
		detectOCFS,
		detectQNX4,
		detectQNX6,
		detectReiser,
		detectReiser4,
		detectROMFS,
		detectSquashFS,
		detectSquashFS3,
		detectSwap,
		detectUBIFS,
		detectUFS,
		detectUFS2,
		detectXFS,
		detectXIAFS,
		detectZFS,
	} {
		if fsType := detect(file); fsType != "" {
			return fsType, nil
		}
	}

	return "", ErrUnknownFilesystem
}

// detectExt detects ext2/ext3/ext4.
func detectExt(file *os.File) string {
	buf := make([]byte, 4096)

	// Read superblock ext at offset 1024.
	file.Seek(1024, 0)
	file.Read(buf)

	// ext magic number is 0xEF53 at offset 56.
	if len(buf) > 57 && binary.LittleEndian.Uint16(buf[56:58]) == 0xEF53 {
		// Read feature flags for versions.
		if len(buf) > 96 {
			features := binary.LittleEndian.Uint32(buf[92:96])
			// If journal (0x0004), must be ext3 o greater.
			if features&0x0004 != 0 {
				// Check for ext4 features.
				if len(buf) > 100 {
					incompat := binary.LittleEndian.Uint32(buf[96:100])
					if incompat&0x0040 != 0 || incompat&0x0080 != 0 {
						return "ext4"
					}
				}
				return "ext3"
			}
		}
		return "ext2"
	}
	return ""
}

// detectXFS detects XFS.
func detectXFS(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// XFS magic number is "XFSB" at offset 0.
	if len(buf) > 4 && string(buf[0:4]) == "XFSB" {
		return "xfs"
	}
	return ""
}

// detectBtrfs detects Btrfs.
func detectBtrfs(file *os.File) string {
	buf := make([]byte, 4096)

	// Btrfs superblock at offset 65536 (64KB).
	file.Seek(65536, 0)
	file.Read(buf)

	// Btrfs magic number is "_BHRfS_M" at offset 64.
	if len(buf) > 72 && string(buf[64:72]) == "_BHRfS_M" {
		return "btrfs"
	}
	return ""
}

// detectNTFS detects NTFS.
func detectNTFS(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// NTFS magic number is "NTFS    " at offset 3.
	if len(buf) > 11 && string(buf[3:11]) == "NTFS    " {
		return "ntfs"
	}
	return ""
}

// detectFAT detects FAT12/FAT16/FAT32.
func detectFAT(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// Verify boot signature 0x55AA at offset 510-511.
	if len(buf) > 511 && buf[510] == 0x55 && buf[511] == 0xAA {
		// FAT32 magic number is "FAT32   " at offset 82.
		if len(buf) > 90 && string(buf[82:90]) == "FAT32   " {
			return "vfat"
		}
		// FAT16/FAT12 magic number is "FAT" at offset 54, or "FAT16" at offset 54.
		if len(buf) > 62 {
			sig := string(buf[54:59])
			if sig[:3] == "FAT" {
				return "vfat"
			}
		}
	}
	return ""
}

// detectSwap detects swap.
func detectSwap(file *os.File) string {
	buf := make([]byte, 4096)

	// Swap signature could be at differents offsets.
	offsets := []int64{4086, 65526} // pagesize-10 for 4K and 64K.

	for _, offset := range offsets {
		file.Seek(offset, 0)
		file.Read(buf[:10])

		if len(buf) >= 10 {
			sig := string(buf[0:10])
			if sig == "SWAPSPACE2" || sig == "SWAP-SPACE" {
				return "swap"
			}
		}
	}
	return ""
}

// detectJFS detects JFS.
func detectJFS(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// JFS magic number "JFS1" at offset 0x200
	if len(buf) > 0x203 && string(buf[0x200:0x204]) == "JFS1" {
		return "jfs"
	}
	return ""
}

// detectReiser detects ReiserFS.
func detectReiser(file *os.File) string {
	buf := make([]byte, 4096)

	// ReiserFS superblock at offset 0x1000.
	file.Seek(0x1000, 0)
	file.Read(buf)

	// ReiserFS magic number at offset 0x38: "ReIsEr"
	if len(buf) > 0x3D && string(buf[0x38:0x3E]) == "ReIsEr" {
		return "reiserfs"
	}
	return ""
}

// detectReiser4 detects Reiser4.
func detectReiser4(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// Reiser4 magic number at offset 0: "ReS4"
	if len(buf) > 4 && string(buf[0:4]) == "ReS4" {
		return "reiser4"
	}
	return ""
}

// detectF2FS detects F2FS.
func detectF2FS(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// F2FS magic number at offset 0x38: 0xF2F52010.
	if len(buf) > 0x3B && string(buf[0x38:0x3C]) == string([]byte{0x10, 0x20, 0xF5, 0xF2}) {
		return "f2fs"
	}
	return ""
}

// detectNILFS detects NILFS2.
func detectNILFS(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// NILFS2 magic number at offset 0x0: 0x3434 ("NILFS")
	if len(buf) > 1 && string(buf[0:2]) == "\x34\x34" {
		return "nilfs2"
	}
	return ""
}

// detectHFSPlusJournaled detects HFS+ with journaling.
func detectHFSPlusJournaled(file *os.File) string {
	buf := make([]byte, 512)

	// HFS+ superblock at offset 1024
	file.Seek(1024, 0)
	file.Read(buf)

	if len(buf) >= 3 {
		sig := string(buf[1:4])
		if sig == "H+J" { // HFS+ Journaled
			return "hfsplusj"
		}
	}
	return ""
}

// detectHFS detects HFS/HFS+.
func detectHFS(file *os.File) string {
	buf := make([]byte, 512)

	// HFS signature at offset 1024: "H+" or "HX".
	file.Seek(1024, 0)
	file.Read(buf)

	if len(buf) > 1 {
		sig := string(buf[1:3])
		if sig == "H+" || sig == "HX" {
			return "hfs"
		}
	}
	return ""
}

// detectExFAT detects exFAT.
func detectExFAT(file *os.File) string {
	buf := make([]byte, 512)

	// exFAT signature at offset 3.
	file.Seek(0x3, 0)
	file.Read(buf)

	if len(buf) > 10 && string(buf[3:11]) == "EXFAT   " {
		return "exfat"
	}
	return ""
}

// detectZFS detects ZFS.
func detectZFS(file *os.File) string {
	buf := make([]byte, 512)
	file.Seek(0, 0)
	file.Read(buf)

	// ZFS uberblock magic at offset 0x38 (little-endian 0x00bab10c)
	if len(buf) > 0x3B && string(buf[0x38:0x3C]) == string([]byte{0x0c, 0xb1, 0xba, 0x00}) {
		return "zfs"
	}
	return ""
}

// detectISO9660 detects ISO9660 (CD/DVD).
func detectISO9660(file *os.File) string {
	buf := make([]byte, 2048)

	// Primary Volume Descriptor at sector 16 (0x8000), +1.
	file.Seek(0x8001, 0)
	file.Read(buf)

	// ISO9660 magic at offset 1: "CD001"
	if len(buf) > 6 && string(buf[1:6]) == "CD001" {
		return "iso9660"
	}
	return ""
}

// detectUDF detects UDF (Universal Disk Format, DVDs/optical media).
func detectUDF(file *os.File) string {
	buf := make([]byte, 8)

	// Primary Volume Descriptor at sector 256 (0x20000), type 1
	file.Seek(0x20000, 0)
	file.Read(buf)

	if len(buf) >= 8 && string(buf[1:6]) == "CD001" {
		return "udf"
	}
	return ""
}

// detectSquashFS detects SquashFS (v4.x).
func detectSquashFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// SquashFS magic 0x73717368 ('sqsh') reversed.
	if len(buf) == 4 && string(buf[0:4]) == "hsqs" {
		return "squashfs"
	}
	return ""
}

// detectSquashFS3 detects SquashFS v3.x.
func detectSquashFS3(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// SquashFS v3 magic 0x73717368.
	if len(buf) == 4 && string(buf[0:4]) == "sqsh" {
		return "squashfs3"
	}
	return ""
}

// detectBcache detects bcache devices.
func detectBcache(file *os.File) string {
	buf := make([]byte, 4096)
	file.Seek(0, 0)
	file.Read(buf)

	// bcache superblock magic at offset 0x0: "BCACHE"
	if len(buf) > 6 && string(buf[0:6]) == "BCACHE" {
		return "bcache"
	}
	return ""
}

// detectBcacheFS detects bcachefs.
func detectBcacheFS(file *os.File) string {
	buf := make([]byte, 16)
	file.Seek(0, 0)
	file.Read(buf)

	// bcachefs magic at offset 0x0: "BCAF"
	if len(buf) > 4 && string(buf[0:4]) == "BCAF" {
		return "bcachefs"
	}
	return ""
}

// detectAPFS detects Apple APFS.
func detectAPFS(file *os.File) string {
	buf := make([]byte, 8)
	file.Seek(0, 0)
	file.Read(buf)

	// APFS container magic "NXSB" at offset 0.
	if len(buf) >= 4 && string(buf[0:4]) == "NXSB" {
		return "apfs"
	}
	return ""
}

// detectEROFS detects EROFS.
func detectEROFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// EROFS magic: 0x5F524F45 ("ERO_")
	if len(buf) >= 4 && string(buf[0:4]) == "ERO\000" {
		return "erofs"
	}
	return ""
}

// detectCramFS detects CramFS.
func detectCramFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// CramFS magic: 0x28cd3d45 (little-endian)
	if len(buf) == 4 && binary.LittleEndian.Uint32(buf) == 0x28cd3d45 {
		return "cramfs"
	}
	return ""
}

// detectBEFS detects BEFS (BeOS filesystem).
func detectBEFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// BEFS magic: "BEFS" at offset 0.
	if len(buf) >= 4 && string(buf[0:4]) == "BEFS" {
		return "befs"
	}
	return ""
}

// detectBFS detects BFS (Boot File System, similar a BeFS).
func detectBFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// BFS magic: "BFS" at offset 0.
	if len(buf) >= 3 && string(buf[0:3]) == "BFS" {
		return "bfs"
	}
	return ""
}

// detectEXFS detects EXFS (experimental / minimal filesystem).
func detectEXFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// EXFS magic: "EXFS" at offset 0
	if len(buf) >= 4 && string(buf[0:4]) == "EXFS" {
		return "exfs"
	}
	return ""
}

// detectGFS detects GFS (Global File System).
func detectGFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// GFS superblock magic: "GFS\0" at offset 0.
	if len(buf) >= 4 && string(buf[0:4]) == "GFS\x00" {
		return "gfs"
	}
	return ""
}

// detectHPFS detects HPFS (High Performance File System).
func detectHPFS(file *os.File) string {
	buf := make([]byte, 8)
	file.Seek(0, 0)
	file.Read(buf)

	// HPFS magic: "HPFS" at offset 3.
	if len(buf) >= 7 && string(buf[3:7]) == "HPFS" {
		return "hpfs"
	}
	return ""
}

// detectMinix detects Minix filesystem.
func detectMinix(file *os.File) string {
	buf := make([]byte, 2)

	// Minix superblock offset is at 0x38.
	file.Seek(0x38, 0)
	file.Read(buf)

	// Minix magic: 0x137F or 0x138F.
	if len(buf) == 2 {
		magic := binary.LittleEndian.Uint16(buf)
		if magic == 0x137F || magic == 0x138F {
			return "minix"
		}
	}
	return ""
}

// detectOCFS detects OCFS (Oracle Clustered FS).
func detectOCFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// OCFS magic: "OCFS" at offset 0.
	if len(buf) >= 4 && string(buf[0:4]) == "OCFS" {
		return "ocfs"
	}
	return ""
}

// detectROMFS detects ROMFS (Read-Only Memory FS).
func detectROMFS(file *os.File) string {
	buf := make([]byte, 4)
	file.Seek(0, 0)
	file.Read(buf)

	// ROMFS magic: "-rom1fs-" at offset 0
	if len(buf) >= 7 && string(buf[0:7]) == "-rom1fs" {
		return "romfs"
	}
	return ""
}

// detectUBIFS detects UBIFS (Unsorted Block Image FS) on flash devices.
func detectUBIFS(file *os.File) string {
	buf := make([]byte, 4)
	// UBIFS superblock magic at start of the volume: 0x06101830
	file.Seek(0, 0)
	file.Read(buf)

	if len(buf) >= 4 && binary.LittleEndian.Uint32(buf) == 0x06101830 {
		return "ubifs"
	}
	return ""
}

// detectJFFS2 detects JFFS2 (Journaling Flash FS).
func detectJFFS2(file *os.File) string {
	buf := make([]byte, 2)
	// JFFS2 magic at start of block: 0x1985
	file.Seek(0, 0)
	file.Read(buf)

	if len(buf) >= 2 && binary.LittleEndian.Uint16(buf) == 0x1985 {
		return "jffs2"
	}
	return ""
}

// detectUFS detects UFS (Unix File System / FFS).
func detectUFS(file *os.File) string {
	buf := make([]byte, 2)
	// UFS superblock magic: 0x011954 or 0x19540119 depending on version
	// Typically at offset 0x400 (1024)
	file.Seek(1024, 0)
	file.Read(buf)

	if len(buf) >= 2 {
		magic := binary.BigEndian.Uint16(buf) // UFS is big-endian
		if magic == 0x1954 || magic == 0x0119 {
			return "ufs"
		}
	}
	return ""
}

// detectUFS2 detects UFS2 (modern version of UFS).
func detectUFS2(file *os.File) string {
	buf := make([]byte, 4)
	// UFS2 superblock magic: 0x19540119 (big endian)
	file.Seek(1024, 0)
	file.Read(buf)

	if len(buf) >= 4 && binary.BigEndian.Uint32(buf) == 0x19540119 {
		return "ufs2"
	}
	return ""
}

// detectXIAFS detects the old Linux XiaFS filesystem.
func detectXIAFS(file *os.File) string {
	buf := make([]byte, 2)
	// XiaFS superblock typically at offset 1024
	file.Seek(1024, 0)
	file.Read(buf)

	if len(buf) >= 2 {
		magic := binary.LittleEndian.Uint16(buf)
		if magic == 0x012F { // XiaFS magic number (historical)
			return "xiafs"
		}
	}
	return ""
}

// detectQNX4 detects QNX4 filesystem.
func detectQNX4(file *os.File) string {
	buf := make([]byte, 4)
	// QNX4 superblock magic at offset 0
	file.Seek(0, 0)
	file.Read(buf)

	if len(buf) >= 4 && string(buf[0:4]) == "QNX4" {
		return "qnx4"
	}
	return ""
}

// detectQNX6 detects QNX6 filesystem.
func detectQNX6(file *os.File) string {
	buf := make([]byte, 4)
	// QNX6 superblock magic at offset 0
	file.Seek(0, 0)
	file.Read(buf)

	if len(buf) >= 4 && string(buf[0:4]) == "QNX6" {
		return "qnx6"
	}
	return ""
}
