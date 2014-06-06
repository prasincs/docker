package network

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/dotcloud/docker/pkg/libcontainer"
)

// Via http://git.kernel.org/cgit/linux/kernel/git/torvalds/linux.git/commit/?id=7b21fddd087678a70ad64afc0f632e0f1071b092
//
// We need different setns values for the different platforms and arch
// We are declaring the macro here because the SETNS syscall does not exist in th stdlib
var setNsMap = map[string]uintptr{
	"linux/amd64": 308,
}

//  crosbymichael: could make a network strategy that instead of returning veth pair names it returns a pid to an existing network namespace
type NetNS struct {
}

func (v *NetNS) Create(n *libcontainer.Network, nspid int, context libcontainer.Context) error {
	context["nspath"] = n.Context["nspath"]
	return nil
}

func (v *NetNS) Initialize(config *libcontainer.Network, context libcontainer.Context) error {
	nspath, exists := context["nspath"]
	if !exists {
		return fmt.Errorf("nspath does not exist in network context")
	}

	f, err := os.OpenFile(nspath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed get network namespace fd: %v", err)
	}

	if err := Setns(f.Fd(), syscall.CLONE_NEWNET); err != nil {
		return fmt.Errorf("failed to setns current network namespace: %v", err)
	}

	return nil
}

func Setns(fd uintptr, flags uintptr) error {
	ns, exists := setNsMap[fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)]
	if !exists {
		return libcontainer.ErrUnsupported
	}

	_, _, err := syscall.RawSyscall(ns, fd, flags, 0)
	if err != 0 {
		return err
	}

	return nil
}
