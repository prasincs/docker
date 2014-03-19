// +build linux

package user

import (
	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/dotcloud/docker/pkg/system"
	"github.com/dotcloud/docker/pkg/user"
	"syscall"
)

// SetupUser sets up the user specified by the container along with the
// correct gid, suid, and groups
func SetupUser(container *libcontainer.Container) error {
	switch container.User {
	case "root", "":
		if err := system.Setgroups(nil); err != nil {
			return err
		}
		if err := system.Setresgid(0, 0, 0); err != nil {
			return err
		}
		if err := system.Setresuid(0, 0, 0); err != nil {
			return err
		}
	default:
		uid, gid, suppGids, err := user.GetUserGroupSupplementary(container.User, syscall.Getuid(), syscall.Getgid())
		if err != nil {
			return err
		}
		if err := system.Setgroups(suppGids); err != nil {
			return err
		}
		if err := system.Setgid(gid); err != nil {
			return err
		}
		if err := system.Setuid(uid); err != nil {
			return err
		}
	}
	return nil
}
