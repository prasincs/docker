package main

import (
	"strconv"
	"syscall"

	"github.com/dotcloud/docker/engine"
)

func (c *container) kill(job *engine.Job) engine.Status {
	var (
		sig, pid int
		err      error
	)

	if pid, err = c.readPid(); err != nil {
		return job.Error(err)
	}
	if sig, err = strconv.Atoi(job.Args[0]); err != nil {
		return job.Error(err)
	}
	if err := syscall.Kill(pid, syscall.Signal(sig)); err != nil {
		return job.Error(err)
	}
	return engine.StatusOK
}
