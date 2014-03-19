// +build linux

package namespaces

import (
	"fmt"
	"github.com/dotcloud/docker/pkg/libcontainer"
	"github.com/dotcloud/docker/pkg/libcontainer/apparmor"
	"github.com/dotcloud/docker/pkg/libcontainer/capabilities"
	"github.com/dotcloud/docker/pkg/libcontainer/network"
	"github.com/dotcloud/docker/pkg/libcontainer/user"
	"github.com/dotcloud/docker/pkg/libcontainer/utils"
	"github.com/dotcloud/docker/pkg/system"
	"os"
	"syscall"
)

// Init is the init process that first runs inside a new namespace to setup mounts, users, networking,
// and other options required for the new container.
func (ns *linuxNs) Init(container *libcontainer.Container, uncleanRootfs, console string, syncPipe *utils.SyncPipe, args []string) error {
	rootfs, err := utils.ResolveRootfs(uncleanRootfs)
	if err != nil {
		return err
	}

	// We always read this as it is a way to sync with the parent as well
	ns.logger.Printf("reading from sync pipe fd %d\n", syncPipe.Child.Fd())
	context, err := syncPipe.ReadFromParent()
	if err != nil {
		syncPipe.Close()
		return err
	}
	ns.logger.Println("received context from parent")
	syncPipe.Close()

	if console != "" {
		ns.logger.Printf("setting up %s as console\n", console)
		slave, err := system.OpenTerminal(console, syscall.O_RDWR)
		if err != nil {
			return fmt.Errorf("open terminal %s", err)
		}
		if err := dupSlave(slave); err != nil {
			return fmt.Errorf("dup2 slave %s", err)
		}
	}
	if _, err := system.Setsid(); err != nil {
		return fmt.Errorf("setsid %s", err)
	}
	if console != "" {
		if err := system.Setctty(); err != nil {
			return fmt.Errorf("setctty %s", err)
		}
	}
	// this is our best effort to let the process know that the parent has died and that it
	// should it should act on it how it sees fit
	if err := system.ParentDeathSignal(uintptr(syscall.SIGTERM)); err != nil {
		return fmt.Errorf("parent death signal %s", err)
	}
	ns.logger.Println("setup mount namespace")
	if err := SetupNewMountNamespace(rootfs, console, container); err != nil {
		return fmt.Errorf("setup mount namespace %s", err)
	}
	if err := network.InitializeNetworks(container, context); err != nil {
		return fmt.Errorf("setup networking %s", err)
	}
	if err := system.Sethostname(container.Hostname); err != nil {
		return fmt.Errorf("sethostname %s", err)
	}
	if err := finalizeNamespace(container); err != nil {
		return fmt.Errorf("finalize namespace %s", err)
	}

	if profile := container.Context["apparmor_profile"]; profile != "" {
		ns.logger.Printf("setting apparmor profile %s\n", profile)
		if err := apparmor.ApplyProfile(os.Getpid(), profile); err != nil {
			return err
		}
	}
	ns.logger.Printf("execing %s\n", args[0])
	return system.Execv(args[0], args[0:], container.Env)
}

// dupSlave dup2 the pty slave's fd into stdout and stdin and ensures that
// the slave's fd is 0, or stdin
func dupSlave(slave *os.File) error {
	if err := system.Dup2(slave.Fd(), 0); err != nil {
		return err
	}
	if err := system.Dup2(slave.Fd(), 1); err != nil {
		return err
	}
	if err := system.Dup2(slave.Fd(), 2); err != nil {
		return err
	}
	return nil
}

// finalizeNamespace drops the caps and sets the correct user
// and working dir before execing the command inside the namespace
func finalizeNamespace(container *libcontainer.Container) error {
	if err := capabilities.DropCapabilities(container); err != nil {
		return fmt.Errorf("drop capabilities %s", err)
	}
	if err := user.SetupUser(container); err != nil {
		return fmt.Errorf("setup user %s", err)
	}
	if container.WorkingDir != "" {
		if err := system.Chdir(container.WorkingDir); err != nil {
			return fmt.Errorf("chdir to %s %s", container.WorkingDir, err)
		}
	}
	return nil
}
