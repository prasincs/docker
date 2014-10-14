package discovery

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// FetchSlaves returns the slaves for the discovery service at the specified endpoint
func FetchSlavesRaw(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	var addrs []string
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&addrs); err != nil {
			return nil, err
		}
	}

	return addrs, nil
}

// FetchSlaves returns the slaves for the discovery service at the specified endpoint
func FetchSlaves(endpoint, username, cluster string) ([]string, error) {
	return FetchSlavesRaw(fmt.Sprintf("%s/u/%s/%s", endpoint, username, cluster))
}

// RegisterSlave adds a new slave identified by the slaveID into the discovery service
// the default TTL is 30 secs
func RegisterSlaveRaw(url, slaveID, addr string) error {
	buf := strings.NewReader(addr)

	_, err := http.Post(fmt.Sprintf("%s/%s", url, slaveID), "application/json", buf)
	return err
}

func RegisterSlave(endpoint, username, cluster, slaveID, addr string) error {
	buf := strings.NewReader(addr)

	_, err := http.Post(fmt.Sprintf("%s/u/%s/%s/%s", endpoint, username, cluster, slaveID), "application/json", buf)
	return err
}
