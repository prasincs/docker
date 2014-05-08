package main

import (
	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/libcontainer/nsinit"
)

func (c *container) init(job *engine.Job) engine.Status {
	var (
		err      error
		syncPipe *nsinit.SyncPipe

		pipe    = job.GetenvInt("pipe")
		console = job.Getenv("console")
		root    = job.Getenv("root")
		// user    = job.Getenv("user")
		// wd      = job.Getenv("wd")
	)
	if syncPipe, err = nsinit.NewSyncPipeFromFd(0, uintptr(pipe)); err != nil {
		return job.Error(err)
	}
	if err := nsinit.Init(c.data, root, console, syncPipe, job.Args); err != nil {
		return job.Error(err)
	}
	return engine.StatusOK
}
