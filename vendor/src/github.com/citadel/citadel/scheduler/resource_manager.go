package scheduler

import (
	"errors"

	"github.com/citadel/citadel"
)

var (
	ErrNoResoucesAvaliable = errors.New("no resources avaliable to schedule container")
)

// ResourceManager is responsible for managing the engines of the cluster
type ResourceManager struct {
}

func NewResourceManager() *ResourceManager {
	return &ResourceManager{}
}

// PlaceImage uses the provided engines to make a decision on which resource the container
// should run based on best utilization of the engines.
func (r *ResourceManager) PlaceContainer(c *citadel.Container, engines []*citadel.State) (*citadel.State, error) {
	scores := []*score{}

	for _, e := range engines {
		if e.Engine.Memory < c.Image.Memory || e.Engine.Cpus < c.Image.Cpus {
			continue
		}

		var (
			cpuScore    = ((e.ReservedCpus() + c.Image.Cpus) / e.Engine.Cpus) * 100.0
			memoryScore = ((e.ReservedMemory() + c.Image.Memory) / e.Engine.Memory) * 100.0
			total       = ((cpuScore + memoryScore) / 200.0) * 100.0
		)

		if cpuScore <= 100.0 && memoryScore <= 100.0 {
			scores = append(scores, &score{r: e, score: total})
		}
	}

	if len(scores) == 0 {
		return nil, ErrNoResoucesAvaliable
	}

	sortScores(scores)

	return scores[0].r, nil
}
