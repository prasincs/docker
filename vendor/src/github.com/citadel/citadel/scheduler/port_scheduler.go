package scheduler

import (
	"github.com/citadel/citadel"
	"log"
)

type PortScheduler struct {
}

func (p *PortScheduler) Schedule(c *citadel.Image, e *citadel.Engine) (bool, error) {
	if len(c.BindPorts) == 0 {
		return true, nil
	}
	for _, port := range c.BindPorts {
		if p.portAlreadyInUse(port, e) {
			return false, nil
		}
	}
	return true, nil
}

func (p *PortScheduler) portAlreadyInUse(port *citadel.Port, e *citadel.Engine) bool {
	state, err := e.State()
	if err != nil {
		return false
	}
	for _, c := range state.Containers {
		for _, cPort := range c.Ports {
			log.Printf("%#v %v %v %v %v %v %v", cPort, cPort.Proto, port.Proto,
				cPort.Port, port.Port, cPort.HostIp, port.HostIp)
			if cPort.Proto == port.Proto && cPort.Port == port.Port {
				// Another container on the same host is binding on the same
				// port/protocol.  Verify if they are requesting the same
				// binding IP, or if the other container is already binding on
				// every IP (0.0.0.0).
				if cPort.HostIp == port.HostIp || cPort.HostIp == "0.0.0.0" {
					return true
				}
			}
		}
	}
	return false
}
