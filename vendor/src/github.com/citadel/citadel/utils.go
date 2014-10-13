package citadel

import (
	"strconv"
	"strings"
	"time"

	"github.com/samalba/dockerclient"
)

type (
	ImageInfo struct {
		Name string
		Tag  string
	}
)

func parsePortInformation(info *dockerclient.ContainerInfo, c *Container) error {
	for pp, b := range info.NetworkSettings.Ports {
		parts := strings.Split(pp, "/")
		rawPort, proto := parts[0], parts[1]

		containerPort, err := strconv.Atoi(rawPort)
		if err != nil {
			return err
		}

		if b == nil {
			c.Ports = append(c.Ports, &Port{
				Proto:         proto,
				ContainerPort: containerPort,
			})
		} else {
			for _, binding := range b {
				port, err := strconv.Atoi(binding.HostPort)
				if err != nil {
					return err
				}

				c.Ports = append(c.Ports, &Port{
					HostIp:        binding.HostIp,
					Proto:         proto,
					Port:          port,
					ContainerPort: containerPort,
				})
			}
		}
	}

	// if we are running in host network mode look at the exposed ports on the image
	// to find out what ports are being exposed
	if info.HostConfig.NetworkMode == "host" {
		for k := range info.Config.ExposedPorts {
			var (
				rawPort string

				parts = strings.Split(k, "/")
				proto = "tcp"
			)

			switch len(parts) {
			case 2:
				rawPort, proto = parts[0], parts[1]
			default:
				rawPort = parts[0]
			}

			port, err := strconv.Atoi(rawPort)
			if err != nil {
				return err
			}

			c.Ports = append(c.Ports, &Port{
				Proto: proto,
				Port:  port,
			})
		}
	}

	return nil
}

func ToDockerContainer(c *Container) dockerclient.Container {
	container := dockerclient.Container{
		Id:      c.ID,
		Names:   []string{"/" + c.Engine.ID + c.Name},
		Created: c.Created,
		Image:   c.Image.Name,
		Command: strings.Join(c.Image.Args, " "),
		Status:  c.State,
	}
	for _, port := range c.Ports {
		port := dockerclient.Port{
			IP:          port.HostIp,
			PrivatePort: port.ContainerPort,
			PublicPort:  port.Port,
			Type:        port.Proto,
		}

		if port.IP == "0.0.0.0" {
			port.IP = c.Engine.IP
		}
		container.Ports = append(container.Ports, port)
	}
	return container
}

func FromDockerContainer(id, image string, engine *Engine) (*Container, error) {
	info, err := engine.client.InspectContainer(id)
	if err != nil {
		return nil, err
	}

	var (
		cType       = ""
		state       = "stopped"
		networkMode = "bridge"
		labels      = map[string]string{}
		env         = make(map[string]string)
	)

	for _, e := range info.Config.Env {
		vals := strings.Split(e, "=")
		k, v := vals[0], vals[1]

		switch k {
		case "_citadel_type":
			cType = v
		case "_citadel_labels":
			for _, tuple := range strings.Split(v, ",") {
				parts := strings.SplitN(tuple, ":", 2)
				if len(parts) == 2 {
					labels[parts[0]] = labels[parts[1]]
				}
			}
		case "HOME", "DEBIAN_FRONTEND", "PATH":
			continue
		default:
			env[k] = v
		}
	}

	if info.State.Running {
		state = "running"
	}

	if m := info.HostConfig.NetworkMode; m != "" {
		networkMode = m
	}
	volDefs := info.Config.Volumes
	vols := []string{}
	for k, _ := range volDefs {
		vols = append(vols, k)
	}

	container := &Container{
		ID:     id,
		Engine: engine,
		Name:   info.Name,
		State:  state,
		Image: &Image{
			Name:        image,
			Args:        info.Config.Cmd,
			Cpus:        float64(info.Config.CpuShares) / 100.0 * engine.Cpus,
			Memory:      float64(info.Config.Memory / 1024 / 1024),
			Volumes:     vols,
			Environment: env,
			Entrypoint:  info.Config.Entrypoint,
			Hostname:    info.Config.Hostname,
			Domainname:  info.Config.Domainname,
			Type:        cType,
			Labels:      labels,
			NetworkMode: networkMode,
			Publish:     info.HostConfig.PublishAllPorts,
			RestartPolicy: RestartPolicy{
				Name:              info.HostConfig.RestartPolicy.Name,
				MaximumRetryCount: info.HostConfig.RestartPolicy.MaximumRetryCount,
			},
		},
	}

	if created, err := time.Parse(time.RFC3339Nano, info.Created); err == nil {
		container.Created = int(created.Unix())
	}

	if err := parsePortInformation(info, container); err != nil {
		return nil, err
	}
	container.Image.BindPorts = container.Ports

	return container, nil
}

func parseImageName(name string) *ImageInfo {
	imageInfo := &ImageInfo{
		Name: name,
		Tag:  "latest",
	}

	img := strings.Split(name, ":")
	if len(img) == 2 {
		imageInfo.Name = img[0]
		imageInfo.Tag = img[1]
	}

	return imageInfo
}
