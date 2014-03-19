package network

import (
	"github.com/dotcloud/docker/pkg/libcontainer"
)

func CreateNetworks(container *libcontainer.Container, nspid int) (libcontainer.Context, error) {
	context := libcontainer.Context{}
	for _, config := range container.Networks {
		strategy, err := GetStrategy(config.Type)
		if err != nil {
			return nil, err
		}
		if err := strategy.Create(config, nspid, context); err != nil {
			return nil, err
		}
	}
	return context, nil
}

// InitializeNetwork uses the network configuration on the container to get the correct
// network strategies to setup the namespaces networking
func InitializeNetworks(container *libcontainer.Container, context libcontainer.Context) error {
	for _, config := range container.Networks {
		strategy, err := GetStrategy(config.Type)
		if err != nil {
			return err
		}

		err1 := strategy.Initialize(config, context)
		if err1 != nil {
			return err1
		}
	}
	return nil
}
