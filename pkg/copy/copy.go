package copy

import (
	"container/list"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/containerd/containerd/pkg/userns"
	"github.com/docker/docker/pkg/pools"
	"github.com/docker/docker/pkg/system"
	"github.com/pkg/xattr"
	"golang.org/x/sys/unix"
)

// Mode indicates whether to use hardlink or copy content
type Mode int

const (
	// Content creates a new file, and copies the content of the file
	Content Mode = iota
	// Hardlink creates a new hardlink to the existing file
	Hardlink
)

func copyRegular(srcPath, dstPath string, fileinfo os.FileInfo) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// If the destination file already exists, we shouldn't blow it away
	dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, fileinfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	return legacyCopy(srcFile, dstFile)
}

func legacyCopy(srcFile io.Reader, dstFile io.Writer) error {
	_, err := pools.Copy(dstFile, srcFile)

	return err
}

func copyXattr(srcPath, dstPath, attr string) error {
	data, err := xattr.LGet(srcPath, attr)
	if err != nil {
		if errors.Is(err, syscall.EOPNOTSUPP) {
			// Task failed successfully: there is no xattr to copy
			// if the source filesystem doesn't support xattrs.
			return nil
		}
		return err
	}
	if data != nil {
		if err := xattr.LSet(dstPath, attr, data); err != nil {
			return err
		}
	}
	return nil
}

type fileID struct {
	dev uint64
	ino uint64
}

type dirMtimeInfo struct {
	dstPath *string
	stat    *syscall.Stat_t
}

// DirCopy copies or hardlinks the contents of one directory to another, properly
// handling soft links, "security.capability" and (optionally) "trusted.overlay.opaque"
// xattrs.
//
// The copyOpaqueXattrs controls if "trusted.overlay.opaque" xattrs are copied.
// Passing false disables copying "trusted.overlay.opaque" xattrs.
func DirCopy(srcDir, dstDir string, copyMode Mode, copyOpaqueXattrs bool) error {

	// This is a map of source file inodes to dst file paths
	copiedFiles := make(map[fileID]string)

	dirsToSetMtimes := list.New()
	err := filepath.Walk(srcDir, func(srcPath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Rebase path
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dstDir, relPath)

		stat, ok := f.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("unable to get raw syscall.Stat_t data for %s", srcPath)
		}

		isHardlink := false

		switch mode := f.Mode(); {
		case mode.IsRegular():
			// the type is 32bit on mips
			id := fileID{dev: uint64(stat.Dev), ino: stat.Ino} //nolint: unconvert
			if copyMode == Hardlink {
				isHardlink = true
				if err2 := os.Link(srcPath, dstPath); err2 != nil {
					return err2
				}
			} else if hardLinkDstPath, ok := copiedFiles[id]; ok {
				if err2 := os.Link(hardLinkDstPath, dstPath); err2 != nil {
					return err2
				}
			} else {
				if err2 := copyRegular(srcPath, dstPath, f); err2 != nil {
					return err2
				}
				copiedFiles[id] = dstPath
			}

		case mode.IsDir():
			if err := os.Mkdir(dstPath, f.Mode()); err != nil && !os.IsExist(err) {
				return err
			}

		case mode&os.ModeSymlink != 0:
			link, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}

			if err := os.Symlink(link, dstPath); err != nil {
				return err
			}

		case mode&os.ModeNamedPipe != 0:
			fallthrough
		case mode&os.ModeSocket != 0:
			if err := unix.Mkfifo(dstPath, uint32(stat.Mode)); err != nil {
				return err
			}

		case mode&os.ModeDevice != 0:
			if userns.RunningInUserNS() {
				// cannot create a device if running in user namespace
				return nil
			}
			if err := unix.Mknod(dstPath, uint32(stat.Mode), int(stat.Rdev)); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unknown file type (%d / %s) for %s", f.Mode(), f.Mode().String(), srcPath)
		}

		// Everything below is copying metadata from src to dst. All this metadata
		// already shares an inode for hardlinks.
		if isHardlink {
			return nil
		}

		if err := os.Lchown(dstPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		copyXattr(srcPath, dstPath, "security.capability") // Ignore error for macOS.

		if copyOpaqueXattrs {
			if err := doCopyXattrs(srcPath, dstPath); err != nil {
				return err
			}
		}

		isSymlink := f.Mode()&os.ModeSymlink != 0

		// There is no LChmod, so ignore mode for symlink. Also, this
		// must happen after chown, as that can modify the file mode
		if !isSymlink {
			if err := os.Chmod(dstPath, f.Mode()); err != nil {
				return err
			}
		}

		// system.Chtimes doesn't support a NOFOLLOW flag atm
		//nolint: unconvert
		if f.IsDir() {
			dirsToSetMtimes.PushFront(&dirMtimeInfo{dstPath: &dstPath, stat: stat})
		} else if !isSymlink {
			aTime := time.Unix(stat.Atimespec.Unix())
			mTime := time.Unix(stat.Mtimespec.Unix())
			if err := system.Chtimes(dstPath, aTime, mTime); err != nil {
				return err
			}
		} else {
			ats := unix.Timespec{Sec: stat.Atimespec.Sec, Nsec: stat.Atimespec.Nsec}
			mts := unix.Timespec{Sec: stat.Mtimespec.Sec, Nsec: stat.Mtimespec.Nsec}
			ts := []unix.Timespec{ats, mts}
			unix.UtimesNano(dstPath, ts) // Ignore error for macOS.
		}
		return nil
	})
	if err != nil {
		return err
	}
	for e := dirsToSetMtimes.Front(); e != nil; e = e.Next() {
		mtimeInfo := e.Value.(*dirMtimeInfo)
		ats := unix.Timespec{Sec: mtimeInfo.stat.Atimespec.Sec, Nsec: mtimeInfo.stat.Atimespec.Nsec}
		mts := unix.Timespec{Sec: mtimeInfo.stat.Mtimespec.Sec, Nsec: mtimeInfo.stat.Mtimespec.Nsec}
		ts := []unix.Timespec{ats, mts}
		unix.UtimesNano(*mtimeInfo.dstPath, ts) // Ignore error for macOS.
	}

	return nil
}

func doCopyXattrs(srcPath, dstPath string) error {
	// We need to copy this attribute if it appears in an overlay upper layer, as
	// this function is used to copy those. It is set by overlay if a directory
	// is removed and then re-created and should not inherit anything from the
	// same dir in the lower dir.
	return copyXattr(srcPath, dstPath, "trusted.overlay.opaque")
}
