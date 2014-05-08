package runtime

import (
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/beam"
)

type Runtime struct {
	ID     string
	Dir    string
	Rootfs string
	Binary string

	graph Graph
}

func (r *Runtime) Run(stdin io.Reader, stdout, stderr io.Writer) error {
	defer func() {
		r.graph.Put(r.ID)
		// deallocate ip
	}()

	cmd := exec.Command(r.Binary, "runtime", r.ID)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "root="+r.Rootfs)

	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}

func (r *Runtime) Job(args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	eng := engine.New()
	eng.Logging = false

	c, err := net.Dial("unix", filepath.Join(r.Dir, "beam.sock"))
	if err != nil {
		return err
	}
	defer c.Close()

	f, err := c.(*net.UnixConn).File()
	if err != nil {
		return err
	}

	child, err := beam.FileConn(f)
	if err != nil {
		return err
	}
	defer child.Close()

	sender := engine.NewSender(child)
	sender.Install(eng)

	job := eng.Job(args[0], args[1:]...)
	job.Stdin.Add(stdin)
	job.Stdout.Add(stdout)
	job.Stderr.Add(stderr)
	job.Env().Import(env)

	return job.Run()
}
