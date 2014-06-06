// +build linux

package console

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/dotcloud/docker/pkg/libcontainer/pkg/label"
	"github.com/dotcloud/docker/pkg/system"
)

// Setup initializes the proper /dev/console inside the rootfs path
func Setup(rootfs, consolePath, mountLabel string) error {
	oldMask := syscall.Umask(0000)
	defer syscall.Umask(oldMask)

	if err := os.Chmod(consolePath, 0600); err != nil {
		return err
	}
	if err := os.Chown(consolePath, 0, 0); err != nil {
		return err
	}
	if err := label.SetFileLabel(consolePath, mountLabel); err != nil {
		return fmt.Errorf("set file label %s %s", consolePath, err)
	}

	dest := filepath.Join(rootfs, "dev/console")

	f, err := os.Create(dest)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("create %s %s", dest, err)
	}
	if f != nil {
		f.Close()
	}

	if err := syscall.Mount(consolePath, dest, "bind", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind %s to %s %s", consolePath, dest, err)
	}
	return nil
}

func OpenAndDup(consolePath string) error {
	slave, err := system.OpenTerminal(consolePath, syscall.O_RDWR)
	if err != nil {
		return fmt.Errorf("open terminal %s", err)
	}
	slaveFd := int(slave.Fd())

	if err := syscall.Dup2(slaveFd, 0); err != nil {
		return err
	}

	if err := syscall.Dup2(slaveFd, 1); err != nil {
		return err
	}

	return syscall.Dup2(slaveFd, 2)
}
