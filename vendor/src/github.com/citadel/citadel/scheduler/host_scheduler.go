package scheduler

import (
	"strings"

	"github.com/citadel/citadel"
)

type HostScheduler struct {
}

func (h *HostScheduler) Schedule(c *citadel.Image, e *citadel.Engine) (bool, error) {
	if len(c.Labels) == 0 {
		return true, nil
	}

	// Perform the check only if the host label was provided.
	if host, ok := c.Labels["host"]; ok {
		return strings.ToLower(host) == strings.ToLower(e.ID), nil
	}

	return true, nil
}
