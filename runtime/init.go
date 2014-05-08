package runtime

import (
	"encoding/json"
	"os"
)

func NewRuntime(id, dir, runtimeBinary string, graph Graph, container interface{}) (*Runtime, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	f, err := os.Create("container.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(container); err != nil {
		return nil, err
	}

	rootfs, err := graph.Get(id)
	if err != nil {
		return nil, err
	}

	return &Runtime{
		ID:     id,
		Dir:    dir,
		Rootfs: rootfs,
		Binary: runtimeBinary,
		graph:  graph,
	}, nil
}
