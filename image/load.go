package image

import (
	"encoding/json"
	"github.com/dotcloud/docker/utils"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

func LoadImage(root string) (*Image, error) {
	jsonData, err := ioutil.ReadFile(jsonPath(root))
	if err != nil {
		return nil, err
	}
	img := &Image{}

	if err := json.Unmarshal(jsonData, img); err != nil {
		return nil, err
	}
	if err := utils.ValidateID(img.ID); err != nil {
		return nil, err
	}
	img.Size, err = getImageSize(root)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func getImageSize(root string) (size int64, err error) {
	if buf, err := ioutil.ReadFile(path.Join(root, "layersize")); err != nil {
		if !os.IsNotExist(err) {
			return -1, err
		}
		// If the layersize file does not exist then set the size to a negative number
		// because a layer size of 0 (zero) is valid
		size = -1
	} else {
		i, err := strconv.Atoi(string(buf))
		if err != nil {
			return -1, err
		}
		size = int64(i)
	}
	return size, nil
}
