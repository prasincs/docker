package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/beam"
	"github.com/dotcloud/docker/pkg/libcontainer"
)

func loadContainer() (*container, error) {
	var data *libcontainer.Container
	f, err := os.Open("container.json")
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	return &container{
		data: data,
	}, nil
}

func runInit(eng *engine.Engine, env []string) {
	job := eng.Job(os.Args[1], os.Args[2:]...)
	job.Env().Init((*engine.Env)(&env))

	job.Setenv("binary", os.Args[0]) // HACK: to get our binaries path

	job.Stdin.Add(os.Stdin)
	job.Stdout.Add(os.Stdout)
	job.Stderr.Add(os.Stderr)

	if err := job.Run(); err != nil {
		fatal(err)
	}
	os.Exit(job.StatusCode())
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	var (
		env = os.Environ()
		eng = engine.New()
	)
	eng.Logging = false

	container, err := loadContainer()
	if err != nil {
		fatal(err)
	}

	if err := container.Install(eng); err != nil {
		fatal(err)
	}

	// HACK: this is so the init process the runs inside the container's namespace
	// and run as a free binary, free of beam right now
	if len(os.Args) > 2 && os.Args[1] == "init" || os.Getenv("FG") != "" {
		runInit(eng, env)
	}

	l, err := net.Listen("unix", "beam.sock")
	if err != nil {
		fatal(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for _ = range sig {
			l.Close()
			os.Remove("beam.sock")
			os.Exit(0)
		}
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}

		go func() {
			defer conn.Close()

			u := conn.(*net.UnixConn)
			beamConn := &beam.UnixConn{UnixConn: u}

			r := engine.NewReceiver(beamConn)
			r.Engine = eng

			if err := r.Run(); err != nil {
				fatal(err)
			}
		}()
	}
}
