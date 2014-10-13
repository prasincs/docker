package docker

import (
	"fmt"
	"time"

	"github.com/citadel/citadel"
	"github.com/citadel/citadel/api"
	"github.com/citadel/citadel/cluster"
	"github.com/citadel/citadel/discovery"
	"github.com/citadel/citadel/scheduler"
)

func Master(url, addr string) error {
	nodes, err := discovery.FetchSlavesRaw(url)
	if err != nil {
		return err
	}

	var engines []*citadel.Engine
	for i, node := range nodes {
		engine := citadel.NewEngine(fmt.Sprintf("node-%d", i), node, 2048, 1)

		if err := engine.Connect(nil); err != nil {
			return err
		}
		engines = append(engines, engine)
	}

	c, err := cluster.New(scheduler.NewResourceManager(), 2*time.Second, engines...)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := c.RegisterScheduler("service", &scheduler.LabelScheduler{}); err != nil {
		return err
	}

	if err := c.Events(api.EventsHandler); err != nil {
		return err
	}
	return api.ListenAndServe(c, addr)
}
