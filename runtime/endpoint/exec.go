package main

import (
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/libcontainer/nsinit"
)

func (c *container) exec(job *engine.Job) engine.Status {
	var (
		root     = job.Getenv("root")
		dataPath = job.Getenv("data_path")
		term     = nsinit.NewTerminal(job.Stdin, job.Stdout, job.Stderr, c.data.Tty)
	)
	defer term.Close()

	// init networking - iptables & proxy - the ip and ports will already be allocated for us
	// TODO: maybe this can be job calls to setup iptables and spawn proxies?

	exit, err := nsinit.Exec(c.data, term, root, dataPath, job.Args, nsinit.DefaultCreateCommand, nil)
	if err != nil {
		return job.Error(err)
	}
	return engine.Status(exit)
}
