package utils

import (
	"encoding/json"
	"fmt"
	"github.com/dotcloud/docker/pkg/libcontainer"
	"io/ioutil"
	"os"
)

// SyncPipe allows communication to and from the Child processes
// to it's Parent and allows the two independent processes to
// syncronize their state.
type SyncPipe struct {
	Parent, Child *os.File
}

func NewSyncPipe() (s *SyncPipe, err error) {
	s = &SyncPipe{}
	s.Child, s.Parent, err = os.Pipe()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func NewSyncPipeFromFd(parendFd, ChildFd uintptr) (*SyncPipe, error) {
	s := &SyncPipe{}
	if parendFd > 0 {
		s.Parent = os.NewFile(parendFd, "parendPipe")
	} else if ChildFd > 0 {
		s.Child = os.NewFile(ChildFd, "ChildPipe")
	} else {
		return nil, fmt.Errorf("no valid sync pipe fd specified")
	}
	return s, nil
}

func (s *SyncPipe) SendToChild(context libcontainer.Context) error {
	data, err := json.Marshal(context)
	if err != nil {
		return err
	}
	s.Parent.Write(data)
	return nil
}

func (s *SyncPipe) ReadFromParent() (libcontainer.Context, error) {
	data, err := ioutil.ReadAll(s.Child)
	if err != nil {
		return nil, fmt.Errorf("error reading from sync pipe %s", err)
	}
	var context libcontainer.Context
	if len(data) > 0 {
		if err := json.Unmarshal(data, &context); err != nil {
			return nil, err
		}
	}
	return context, nil

}

func (s *SyncPipe) Close() error {
	if s.Parent != nil {
		s.Parent.Close()
	}
	if s.Child != nil {
		s.Child.Close()
	}
	return nil
}
