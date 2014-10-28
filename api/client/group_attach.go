package client

import (
	"bufio"
	"fmt"
	"io"

	"github.com/docker/docker/api"
)

type attachEvent struct {
	container *api.Container
	line      string
	status    string
	err       error
}

func (cli *DockerCli) attachToGroup(group *api.Group) error {
	events := make(chan attachEvent)
	open := make(map[*api.Container]bool)

	color, prefix := getColorsAndPrefixes(group.Containers)

	for _, c := range group.Containers {
		open[c] = true
		cli.attachToGroupContainer(group, c, events)
	}

	for e := range events {
		if e.err != nil {
			if e.err != io.EOF {
				fmt.Fprintf(cli.out, "%s%v\n", prefix[e.container], e.err)
			}
			delete(open, e.container)
		} else if e.status != "" {
			fmt.Fprintf(cli.out, "%s%s\n", prefix[e.container], color[e.container](e.status))
		} else {
			fmt.Fprintf(cli.out, "%s%s", prefix[e.container], e.line)
		}

		if len(open) <= 0 {
			break
		}
	}

	return nil
}

func (cli *DockerCli) attachToGroupContainer(group *api.Group, container *api.Container, events chan attachEvent) {
	r, w := io.Pipe()

	go func() {
		status, err := waitForExit(cli, fmt.Sprintf("%s/%s", group.Name, container.Name))
		if err != nil {
			return // never mind, we just won't display the exit status
		}
		events <- attachEvent{
			container: container,
			status:    fmt.Sprintf("exited with status %d", status),
		}
	}()

	go func() {
		path := fmt.Sprintf("/containers/%s/%s/attach?stream=1&logs=1&stdin=0&stdout=1&stderr=1", group.Name, container.Name)
		if err := cli.hijack("POST", path, false, nil, w, w, nil, nil); err != nil {
			w.CloseWithError(err)
		}
		w.Close()
	}()

	go func() {
		reader := bufio.NewReader(r)
		for {
			line, err := reader.ReadString('\n')
			events <- attachEvent{
				container: container,
				line:      line,
				err:       err,
			}
			if err != nil {
				break
			}
		}
	}()
}

func getColorsAndPrefixes(containers []*api.Container) (map[*api.Container]func(string) string, map[*api.Container]string) {
	prefixLen := 0
	for _, c := range containers {
		if len(c.Name) > prefixLen {
			prefixLen = len(c.Name)
		}
	}
	prefixFmt := fmt.Sprintf("%%-%ds | ", prefixLen)

	cycle := []int{36, 33, 32, 35, 31, 34}
	color := make(map[*api.Container]func(string) string)
	prefix := make(map[*api.Container]string)

	for i, c := range containers {
		color[c] = ansiFn(cycle[i%len(cycle)])
		prefix[c] = color[c](fmt.Sprintf(prefixFmt, c.Name))
	}

	return color, prefix
}

func ansiFn(code int) func(string) string {
	return func(str string) string {
		return fmt.Sprintf("\033[%d;m%s\033[0m", code, str)
	}
}
