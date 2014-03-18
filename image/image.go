package image

import (
	"encoding/json"
	"fmt"
	"github.com/dotcloud/docker/archive"
	"github.com/dotcloud/docker/runconfig"
	"github.com/dotcloud/docker/runtime/graphdriver"
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"path"
	"strconv"
	"time"
)

type Image struct {
	ID              string            `json:"id"`
	Parent          string            `json:"parent,omitempty"`
	Comment         string            `json:"comment,omitempty"`
	Created         time.Time         `json:"created"`
	Container       string            `json:"container,omitempty"`
	ContainerConfig runconfig.Config  `json:"container_config,omitempty"`
	DockerVersion   string            `json:"docker_version,omitempty"`
	Author          string            `json:"author,omitempty"`
	Config          *runconfig.Config `json:"config,omitempty"`
	Architecture    string            `json:"architecture,omitempty"`
	OS              string            `json:"os,omitempty"`
	Size            int64             `json:"Size,omitempty"` // FIXME: we messed up the case on this field

	graph Graph
}

// Build an Image object from raw json data
func NewImgJSON(src []byte) (img *Image, err error) {
	if err := json.Unmarshal(src, &img); err != nil {
		return nil, err
	}
	return img, nil
}

func (img *Image) SetGraph(graph Graph) {
	img.graph = graph
}

// SaveSize stores the current `size` value of `img` in the directory `root`.
func (img *Image) SaveSize(root string) error {
	if err := ioutil.WriteFile(path.Join(root, "layersize"), []byte(strconv.Itoa(int(img.Size))), 0600); err != nil {
		return fmt.Errorf("Error storing image size in %s/layersize: %s", root, err)
	}
	return nil
}

// TarLayer returns a tar archive of the image's filesystem layer.
func (img *Image) TarLayer() (arch archive.Archive, err error) {
	if img.graph == nil {
		return nil, fmt.Errorf("Can't load storage driver for unregistered image %s", img.ID)
	}
	driver := img.graph.Driver()
	if differ, ok := driver.(graphdriver.Differ); ok {
		return differ.Diff(img.ID)
	}

	imgFs, err := driver.Get(img.ID)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			driver.Put(img.ID)
		}
	}()

	if img.Parent == "" {
		archive, err := archive.Tar(imgFs, archive.Uncompressed)
		if err != nil {
			return nil, err
		}
		return utils.NewReadCloserWrapper(archive, func() error {
			err := archive.Close()
			driver.Put(img.ID)
			return err
		}), nil
	}

	parentFs, err := driver.Get(img.Parent)
	if err != nil {
		return nil, err
	}
	defer driver.Put(img.Parent)

	changes, err := archive.ChangesDirs(imgFs, parentFs)
	if err != nil {
		return nil, err
	}
	archive, err := archive.ExportChanges(imgFs, changes)
	if err != nil {
		return nil, err
	}
	return utils.NewReadCloserWrapper(archive, func() error {
		err := archive.Close()
		driver.Put(img.ID)
		return err
	}), nil
}

// Image includes convenience proxy functions to its graph
// These functions will return an error if the image is not registered
// (ie. if image.graph == nil)
func (img *Image) History() ([]*Image, error) {
	var parents []*Image
	if err := img.WalkHistory(
		func(img *Image) error {
			parents = append(parents, img)
			return nil
		},
	); err != nil {
		return nil, err
	}
	return parents, nil
}

func (img *Image) WalkHistory(handler func(*Image) error) (err error) {
	currentImg := img
	for currentImg != nil {
		if handler != nil {
			if err := handler(currentImg); err != nil {
				return err
			}
		}
		currentImg, err = currentImg.GetParent()
		if err != nil {
			return fmt.Errorf("Error while getting parent image: %v", err)
		}
	}
	return nil
}

func (img *Image) GetParent() (*Image, error) {
	if img.Parent == "" {
		return nil, nil
	}
	if img.graph == nil {
		return nil, fmt.Errorf("Can't lookup parent of unregistered image")
	}
	return img.graph.Get(img.Parent)
}

func (img *Image) root() (string, error) {
	if img.graph == nil {
		return "", fmt.Errorf("Can't lookup root of unregistered image")
	}
	return img.graph.ImageRoot(img.ID), nil
}

func (img *Image) GetParentsSize(size int64) int64 {
	parentImage, err := img.GetParent()
	if err != nil || parentImage == nil {
		return size
	}
	size += parentImage.Size
	return parentImage.GetParentsSize(size)
}

// Depth returns the number of parents for a
// current image
func (img *Image) Depth() (int, error) {
	var (
		count  = 0
		parent = img
		err    error
	)

	for parent != nil {
		count++
		parent, err = parent.GetParent()
		if err != nil {
			return -1, err
		}
	}
	return count, nil
}
