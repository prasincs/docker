package main

import (
	"io/ioutil"
	"strconv"

	"github.com/dotcloud/docker/engine"
	"github.com/dotcloud/docker/pkg/libcontainer"
)

type container struct {
	data *libcontainer.Container
}

func (c *container) Install(eng *engine.Engine) error {
	for name, handler := range map[string]engine.Handler{
		"init": c.init,
		"exec": c.exec,
		"kill": c.kill,
	} {
		if err := eng.Register(name, handler); err != nil {
			return err
		}
	}
	return nil
}

func (c *container) readPid() (int, error) {
	data, err := ioutil.ReadFile("pid")
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(string(data))
}
