package links

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/docker/docker/engine"
	"github.com/docker/docker/nat"
	"github.com/docker/docker/reexec"
	"github.com/docker/libcontainer/network"
	"github.com/docker/libcontainer/system"
)

var root = "/var/lib/docker/links"

func init() {
	reexec.Register("docker-createns", createns)
	reexec.Register("docker-setupns", setupns)
}

type Link struct {
	ParentIP         string
	ChildIP          string
	Name             string
	ChildEnvironment []string
	Ports            []nat.Port
	IsEnabled        bool
	eng              *engine.Engine
}

func CreateSharedLink(name string) error {
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(root, name)); err == nil {
		return nil
	}

	cmd := &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{
			"docker-createns",
			"-path", filepath.Join(root, name),
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	// configure new netns
	if err := network.CreateVethPair("vethmommy", "vethbabby"); err != nil {
		return err
	}

	f, err := os.Open(filepath.Join(root, name))
	if err != nil {
		return err
	}

	if err := network.SetInterfaceMaster("vethmommy", "docker0"); err != nil {
		return err
	}

	if err := network.InterfaceUp("vethmommy"); err != nil {
		return err
	}

	if err := network.SetInterfaceInNamespaceFd("vethbabby", f.Fd()); err != nil {
		return err
	}

	cmd = &exec.Cmd{
		Path: reexec.Self(),
		Args: []string{
			"docker-setupns",
		},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	return cmd.Run()
}

func GetSharedLink(name string) string {
	return filepath.Join(root, name)
}

func setupns() {
	runtime.LockOSThread()
	f, err := os.Open("/var/lib/docker/links/redis")
	if err != nil {
		log.Fatal(err)
	}

	if err := system.Setns(f.Fd(), 0); err != nil {
		log.Fatal(err)
	}

	if err := network.InterfaceDown("vethbabby"); err != nil {
		log.Fatal(err)
	}

	if err := network.ChangeInterfaceName("vethbabby", "redis0"); err != nil {
		log.Fatal(err)
	}

	if err := network.SetInterfaceIp("redis0", "10.0.42.101/16"); err != nil {
		log.Fatal(err)
	}

	if err := network.InterfaceUp("redis0"); err != nil {
		log.Fatal(err)
	}

	if err := network.SetDefaultGateway("10.0.42.1", "redis0"); err != nil {
		log.Fatal(err)
	}

	if err := network.InterfaceUp("lo"); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

func createns() {
	runtime.LockOSThread()

	path := flag.String("path", "", "path to bind mount file")
	flag.Parse()

	if err := syscall.Unshare(syscall.CLONE_NEWNET); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(*path)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	if f != nil {
		f.Close()
	}

	if err := syscall.Mount("/proc/self/ns/net", *path, "bind", syscall.MS_BIND, ""); err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}

func NewLink(parentIP, childIP, name string, env []string, exposedPorts map[nat.Port]struct{}, eng *engine.Engine) (*Link, error) {

	var (
		i     int
		ports = make([]nat.Port, len(exposedPorts))
	)

	for p := range exposedPorts {
		ports[i] = p
		i++
	}

	l := &Link{
		Name:             name,
		ChildIP:          childIP,
		ParentIP:         parentIP,
		ChildEnvironment: env,
		Ports:            ports,
		eng:              eng,
	}
	return l, nil

}

func (l *Link) Alias() string {
	_, alias := path.Split(l.Name)
	return alias
}

func (l *Link) ToEnv() []string {
	env := []string{}
	alias := strings.Replace(strings.ToUpper(l.Alias()), "-", "_", -1)

	if p := l.getDefaultPort(); p != nil {
		env = append(env, fmt.Sprintf("%s_PORT=%s://%s:%s", alias, p.Proto(), l.ChildIP, p.Port()))
	}

	// Load exposed ports into the environment
	for _, p := range l.Ports {
		env = append(env, fmt.Sprintf("%s_PORT_%s_%s=%s://%s:%s", alias, p.Port(), strings.ToUpper(p.Proto()), p.Proto(), l.ChildIP, p.Port()))
		env = append(env, fmt.Sprintf("%s_PORT_%s_%s_ADDR=%s", alias, p.Port(), strings.ToUpper(p.Proto()), l.ChildIP))
		env = append(env, fmt.Sprintf("%s_PORT_%s_%s_PORT=%s", alias, p.Port(), strings.ToUpper(p.Proto()), p.Port()))
		env = append(env, fmt.Sprintf("%s_PORT_%s_%s_PROTO=%s", alias, p.Port(), strings.ToUpper(p.Proto()), p.Proto()))
	}

	// Load the linked container's name into the environment
	env = append(env, fmt.Sprintf("%s_NAME=%s", alias, l.Name))

	if l.ChildEnvironment != nil {
		for _, v := range l.ChildEnvironment {
			parts := strings.Split(v, "=")
			if len(parts) != 2 {
				continue
			}
			// Ignore a few variables that are added during docker build (and not really relevant to linked containers)
			if parts[0] == "HOME" || parts[0] == "PATH" {
				continue
			}
			env = append(env, fmt.Sprintf("%s_ENV_%s=%s", alias, parts[0], parts[1]))
		}
	}
	return env
}

// Default port rules
func (l *Link) getDefaultPort() *nat.Port {
	var p nat.Port
	i := len(l.Ports)

	if i == 0 {
		return nil
	} else if i > 1 {
		nat.Sort(l.Ports, func(ip, jp nat.Port) bool {
			// If the two ports have the same number, tcp takes priority
			// Sort in desc order
			return ip.Int() < jp.Int() || (ip.Int() == jp.Int() && strings.ToLower(ip.Proto()) == "tcp")
		})
	}
	p = l.Ports[0]
	return &p
}

func (l *Link) Enable() error {
	if err := l.toggle("-I", false); err != nil {
		return err
	}
	l.IsEnabled = true
	return nil
}

func (l *Link) Disable() {
	// We do not care about errors here because the link may not
	// exist in iptables
	l.toggle("-D", true)

	l.IsEnabled = false
}

func (l *Link) toggle(action string, ignoreErrors bool) error {
	job := l.eng.Job("link", action)

	job.Setenv("ParentIP", l.ParentIP)
	job.Setenv("ChildIP", l.ChildIP)
	job.SetenvBool("IgnoreErrors", ignoreErrors)

	out := make([]string, len(l.Ports))
	for i, p := range l.Ports {
		out[i] = fmt.Sprintf("%s/%s", p.Port(), p.Proto())
	}
	job.SetenvList("Ports", out)

	if err := job.Run(); err != nil {
		// TODO: get ouput from job
		return err
	}
	return nil
}
