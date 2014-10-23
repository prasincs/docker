package citadel

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/samalba/dockerclient"
)

func NewEngine(id string, addr string, cpus float64, memory float64) *Engine {
	e := &Engine{
		ID:     id,
		Addr:   addr,
		Cpus:   cpus,
		Memory: memory,
		Labels: make(map[string]string),
		ch:     make(chan bool),
	}
	return e
}

type Engine struct {
	ID     string            `json:"id,omitempty"`
	IP     string            `json:"ip,omitempty"`
	Addr   string            `json:"addr,omitempty"`
	Cpus   float64           `json:"cpus,omitempty"`
	Memory float64           `json:"memory,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`

	mux          sync.Mutex
	ch           chan bool
	state        *State
	client       *dockerclient.DockerClient
	eventHandler EventHandler
}

func (e *Engine) Connect(config *tls.Config) error {
	c, err := dockerclient.NewDockerClient(e.Addr, config)
	if err != nil {
		return err
	}

	addr, err := net.ResolveIPAddr("ip4", strings.Split(c.URL.Host, ":")[0])
	if err != nil {
		return err
	}
	e.IP = addr.IP.String()

	e.client = c

	// Fetch the engine labels.
	if err := e.fetchLabels(); err != nil {
		e.client = nil
		return err
	}

	// Force a state update before returning.
	if err := e.updateState(); err != nil {
		e.client = nil
		return err
	}

	// Start the update loop.
	go e.updateLoop()

	// Start monitoring events from the Engine.
	e.client.StartMonitorEvents(e.handler)

	return nil
}

func (e *Engine) fetchLabels() error {
	info, err := e.client.Info()
	if err != nil {
		return err
	}
	e.Labels["driver"] = info.Driver
	e.Labels["executiondriver"] = info.ExecutionDriver
	e.Labels["kernelversion"] = info.KernelVersion
	e.Labels["operatingsystem"] = info.OperatingSystem
	return nil
}

func (e *Engine) updateLoop() {
	for {
		var err error
		select {
		case <-e.ch:
			err = e.updateState()
		case <-time.After(30 * time.Second):
			err = e.updateState()
		}
		if err != nil {
			log.Printf("[%s] Updated state failed: %v", e.ID, err)
		}
	}
}

func (e *Engine) SetClient(c *dockerclient.DockerClient) {
	e.client = c
}

// IsConnected returns true if the engine is connected to a remote docker API
func (e *Engine) IsConnected() bool {
	return e.client != nil
}

func (e *Engine) Pull(image string) error {
	imageInfo := parseImageName(image)
	if err := e.client.PullImage(imageInfo.Name, imageInfo.Tag); err != nil {
		return err
	}
	return nil
}

func (e *Engine) Create(c *Container, pullImage bool) error {
	var (
		err    error
		env    = []string{}
		client = e.client
		i      = c.Image
	)
	c.Engine = e

	for k, v := range i.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	labels := []string{}
	for k, v := range i.Labels {
		labels = append(labels, fmt.Sprintf("%s:%s", k, v))
	}

	env = append(env,
		fmt.Sprintf("_citadel_type=%s", i.Type),
		fmt.Sprintf("_citadel_labels=%s", strings.Join(labels, ",")),
	)

	vols := make(map[string]struct{})
	for _, v := range i.Volumes {
		if strings.Index(v, ":") > -1 {
			cv := strings.Split(v, ":")
			v = cv[1]
		}
		vols[v] = struct{}{}
	}

	binds := []string{}
	for _, v := range c.Image.Volumes {
		if strings.Index(v, ":") > -1 {
			binds = append(binds, v)
		}
	}

	links := []string{}
	for k, v := range c.Image.Links {
		links = append(links, fmt.Sprintf("%s:%s", k, v))
	}

	config := &dockerclient.ContainerConfig{
		Hostname:     i.Hostname,
		Domainname:   i.Domainname,
		Image:        i.Name,
		Cmd:          i.Args,
		Memory:       int(i.Memory) * 1024 * 1024,
		Env:          env,
		CpuShares:    int(i.Cpus * 100.0 / e.Cpus),
		ExposedPorts: make(map[string]struct{}),
		Volumes:      vols,
		HostConfig: dockerclient.HostConfig{
			PortBindings:    make(map[string][]dockerclient.PortBinding),
			PublishAllPorts: i.Publish,
			Links:           links,
			Binds:           binds,
			RestartPolicy: dockerclient.RestartPolicy{
				Name:              c.Image.RestartPolicy.Name,
				MaximumRetryCount: c.Image.RestartPolicy.MaximumRetryCount,
			},
			NetworkMode: c.Image.NetworkMode,
		},
	}

	for _, b := range i.BindPorts {
		key := fmt.Sprintf("%d/%s", b.ContainerPort, b.Proto)
		config.ExposedPorts[key] = struct{}{}
		if _, ok := config.HostConfig.PortBindings[key]; !ok {
			config.HostConfig.PortBindings[key] = []dockerclient.PortBinding{}
		}
		config.HostConfig.PortBindings[key] = append(config.HostConfig.PortBindings[key], dockerclient.PortBinding{
			HostIp:   b.HostIp,
			HostPort: fmt.Sprint(b.Port),
		})
	}
	for _, b := range i.ExposedPorts {
		config.ExposedPorts[b] = struct{}{}
	}

	if pullImage {
		if err := e.Pull(i.Name); err != nil {
			return err
		}
	}

	if c.ID, err = client.CreateContainer(config, c.Name); err != nil {
		// If the error is other than not found, abort immediately.
		if err != dockerclient.ErrNotFound {
			return err
		}
		// Otherwise, try to pull the image...
		if err = e.Pull(i.Name); err != nil {
			return err
		}
		// ...And try again.
		if c.ID, err = client.CreateContainer(config, c.Name); err != nil {
			return err
		}
	}

	// Set the state to pending immediately.
	c.State = "pending"
	c.Created = int(time.Now().Unix())

	if err = e.updatePortInformation(c); err != nil {
		return err
	}

	// Register the container immediately while waiting for a state refresh.
	e.mux.Lock()
	defer e.mux.Unlock()
	e.state.Containers[c.ID] = c

	return nil
}

func (e *Engine) Start(c *Container) error {
	return e.client.StartContainer(c.ID, nil)
}

func (e *Engine) ListImages() ([]string, error) {
	images, err := e.client.ListImages()
	if err != nil {
		return nil, err
	}

	out := []string{}

	for _, i := range images {
		for _, t := range i.RepoTags {
			out = append(out, t)
		}
	}

	return out, nil
}

func (e *Engine) updatePortInformation(c *Container) error {
	info, err := e.client.InspectContainer(c.ID)
	if err != nil {
		return err
	}

	return parsePortInformation(info, c)
}

func (e *Engine) ListContainers(all bool) ([]*Container, error) {
	out := []*Container{}

	c, err := e.client.ListContainers(all)
	if err != nil {
		return nil, err
	}

	for _, ci := range c {
		cc, err := FromDockerContainer(ci.Id, ci.Image, e)
		if err != nil {
			return nil, err
		}

		out = append(out, cc)
	}

	return out, nil
}

func (e *Engine) Kill(container *Container, sig int) error {
	return e.client.KillContainer(container.ID)
}

func (e *Engine) Stop(container *Container) error {
	return e.client.StopContainer(container.ID, 8)
}

func (e *Engine) Restart(container *Container, timeout int) error {
	return e.client.RestartContainer(container.ID, timeout)
}

func (e *Engine) Remove(container *Container, force bool) error {
	if err := e.client.RemoveContainer(container.ID, force); err != nil {
		return err
	}

	// Remove the container from the state. Eventually, the state refresh loop
	// will rewrite this.
	e.mux.Lock()
	defer e.mux.Unlock()
	delete(e.state.Containers, container.ID)

	return nil
}

func (e *Engine) Events(h EventHandler) error {
	if e.eventHandler != nil {
		return fmt.Errorf("event handler already set")
	}
	e.eventHandler = h
	return nil
}

func (e *Engine) updateState() error {
	containers, err := e.ListContainers(true)
	if err != nil {
		return err
	}

	e.mux.Lock()
	defer e.mux.Unlock()

	e.state = &State{
		Engine:     e,
		Containers: make(map[string]*Container),
	}

	for _, c := range containers {
		e.state.Containers[c.ID] = c
	}

	log.Printf("[%s] Updated state", e.ID)

	return nil
}

func (e *Engine) updateStateAsync() {
	e.ch <- true
}

func (e *Engine) State() (*State, error) {
	return e.state, nil
}

func (e *Engine) String() string {
	return fmt.Sprintf("engine %s addr %s", e.ID, e.Addr)
}

func (e *Engine) handler(ev *dockerclient.Event, args ...interface{}) {
	// Something changed - refresh our internal state.
	e.updateStateAsync()

	// If there is no event handler registered, abort right now.
	if e.eventHandler == nil {
		return
	}

	event := &Event{
		Engine: e,
		Type:   ev.Status,
		Time:   time.Unix(int64(ev.Time), 0),
	}

	container, err := FromDockerContainer(ev.Id, ev.From, e)
	if err != nil {
		// TODO: un fuck this shit, fuckin handler
		return
	}

	event.Container = container

	e.eventHandler.Handle(event)
}
