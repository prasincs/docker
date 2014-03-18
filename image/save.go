package image

import (
	"encoding/json"
	"github.com/dotcloud/docker/archive"
	"github.com/dotcloud/docker/runtime/graphdriver"
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"os"
	"path"
	"time"
)

func StoreImage(img *Image, jsonData []byte, layerData archive.ArchiveReader, root, layer string) error {
	if err := os.MkdirAll(layer, 0755); err != nil {
		return err
	}
	if err := unpackLayer(img, layerData, layer); err != nil {
		return err
	}
	if err := img.SaveSize(root); err != nil {
		return err
	}
	return saveConfigruation(img, jsonData, root)
}

func unpackLayer(img *Image, layerData archive.ArchiveReader, layer string) error {
	var (
		size   int64
		err    error
		driver = img.graph.Driver()
	)

	// If layerData is not nil, unpack it into the new layer
	if layerData != nil {
		if differ, ok := driver.(graphdriver.Differ); ok {
			if err := differ.ApplyDiff(img.ID, layerData); err != nil {
				return err
			}
			if size, err = differ.DiffSize(img.ID); err != nil {
				return err
			}
		} else {
			start := time.Now().UTC()
			utils.Debugf("Start untar layer")
			if err := archive.ApplyLayer(layer, layerData); err != nil {
				return err
			}
			utils.Debugf("Untar time: %vs", time.Now().UTC().Sub(start).Seconds())

			if img.Parent == "" {
				if size, err = utils.TreeSize(layer); err != nil {
					return err
				}
			} else {
				parent, err := driver.Get(img.Parent)
				if err != nil {
					return err
				}
				defer driver.Put(img.Parent)

				changes, err := archive.ChangesDirs(layer, parent)
				if err != nil {
					return err
				}
				size = archive.ChangesSize(layer, changes)
			}
		}
	}

	img.Size = size
	return nil
}

func saveConfigruation(img *Image, jsonData []byte, root string) error {
	var err error
	// If raw json is provided, then use it
	if jsonData == nil {
		if jsonData, err = json.Marshal(img); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(jsonPath(root), jsonData, 0600)
}

func jsonPath(root string) string {
	return path.Join(root, "json")
}
