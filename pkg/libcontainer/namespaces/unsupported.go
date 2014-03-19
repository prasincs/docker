// +build !linux

package namespaces

import (
	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/dotcloud/docker/pkg/libcontainer/utils"
)

func (ns *linuxNs) Exec(container *libcontainer.Container, term Terminal, args []string) (int, error) {
	return -1, libcontainer.ErrUnsupported
}

func (ns *linuxNs) ExecIn(container *libcontainer.Container, nspid int, args []string) (int, error) {
	return -1, libcontainer.ErrUnsupported
}

func (ns *linuxNs) Init(container *libcontainer.Container, uncleanRootfs, console string, syncPipe *utils.SyncPipe, args []string) error {
	return libcontainer.ErrUnsupported
}
