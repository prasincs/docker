package scheduler

import (
	"fmt"
	"strings"

	"github.com/citadel/citadel"
)

// UniqueScheduler only returns engines that do not have the image running
type UniqueScheduler struct {
}

func (u *UniqueScheduler) Schedule(c *citadel.Image, e *citadel.Engine) (bool, error) {
	containers, err := e.ListContainers(false)
	if err != nil {
		return false, err
	}

	if u.hasImage(c, containers) {
		return false, nil
	}

	return true, nil
}

func (u *UniqueScheduler) hasImage(i *citadel.Image, containers []*citadel.Container) bool {
	fullImage := i.Name

	if !strings.Contains(fullImage, ":") {
		fullImage = fmt.Sprintf("%s:latest", fullImage)
	}

	for _, c := range containers {
		if c.Image.Name == fullImage {
			return true
		}
	}

	return false
}
