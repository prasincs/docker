package docker

import (
	"crypto/md5"
	"fmt"
	"log"
	"time"

	"github.com/citadel/citadel"
	"github.com/citadel/citadel/api"
	"github.com/citadel/citadel/cluster"
	"github.com/citadel/citadel/discovery"
	"github.com/citadel/citadel/scheduler"
)

func Master(url, addr string) error {
	c, err := cluster.New(scheduler.NewResourceManager())
	if err != nil {
		return err
	}
	defer c.Close()

	scheduler := scheduler.NewMultiScheduler(
		&scheduler.HostScheduler{},
		&scheduler.PortScheduler{},
		&scheduler.LabelScheduler{})
	if err := c.RegisterScheduler("service", scheduler); err != nil {
		return err
	}

	go func() {
		nodes, err := discovery.FetchSlavesRaw(url)
		if err == nil {
			for _, node := range nodes {
				found := false
				for _, e := range c.Engines() {
					if e.Addr == node {
						found = true
						break
					}
				}

				if !found {
					engine := citadel.NewEngine(fmt.Sprintf("node-%x", md5.Sum([]byte(node))), node, 1, 2048)

					if err := engine.Connect(nil); err == nil {
						log.Println("Adding new node:", engine.ID)
						c.AddEngine(engine)
						engine.Events(api.EventsHandler)
					}
				}
			}
		} else {
			log.Printf("[error] %v\n", err)
		}
		time.Sleep(5 * time.Second) // very low timeout for the demo
	}()

	return api.ListenAndServe(c, addr)
}
