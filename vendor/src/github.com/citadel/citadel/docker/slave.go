package docker

import (
	"time"

	"github.com/citadel/citadel/discovery"
)

func Slave(url, slaveID, addr string) error {
	if err := discovery.RegisterSlaveRaw(url, slaveID, addr); err != nil {
		return err
	}

	// heartbeat every 25 seconds
	go func() {
		for {
			time.Sleep(25 * time.Second)
			discovery.RegisterSlaveRaw(url, slaveID, addr)
		}
	}()
	return nil
}
