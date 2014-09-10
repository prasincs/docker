package coredata

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type Coredata struct {
	sync.Mutex

	root string
}

func New(path string) (*Coredata, error) {
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}

	return &Coredata{
		root: path,
	}, nil
}

func (d *Coredata) Close() error {
	return nil
}

func (d *Coredata) CreateGroup(name string) error {
	return os.MkdirAll(d.join("/groups", name), 0700)
}

func (d *Coredata) RemoveGroup(name string) error {
	return os.RemoveAll(d.join("/groups", name))
}

func (d *Coredata) GroupExists(name string) (bool, error) {
	_, err := os.Stat(d.join("/groups", name))

	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (d *Coredata) ListGroups() ([]string, error) {
	return d.ls("/groups")
}

func (d *Coredata) ListContainersInGroup(groupName string) ([]string, error) {
	return d.ls("/groups", groupName, "containers")
}

func (d *Coredata) AddContainerToGroup(groupName, containerID string) error {
	return os.MkdirAll(d.join("/groups", groupName, "containers", containerID), 0700)
}

func (d *Coredata) RemoveContainerFromGroup(groupName, containerID string) error {
	return os.RemoveAll(d.join("/groups", groupName, "containers", containerID))
}

func (d *Coredata) ls(args ...string) ([]string, error) {
	var out []string

	fi, err := ioutil.ReadDir(d.join(args...))
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}

		return nil, err
	}

	for _, f := range fi {
		out = append(out, f.Name())
	}

	return out, nil
}

func (d *Coredata) join(args ...string) string {
	return filepath.Join(append([]string{d.root}, args...)...)
}
