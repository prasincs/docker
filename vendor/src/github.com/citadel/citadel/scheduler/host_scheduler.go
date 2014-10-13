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

	return strings.ToLower(c.Labels["host"]) == strings.ToLower(e.ID), nil
}
