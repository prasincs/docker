package discovery

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

var (
	ErrNotFound = errors.New("cluster not found")
)

// Client handles the server side connection to the discovery datastore
type Client struct {
	client *etcd.Client
}

type slaveMetadata struct {
	Addr string `json:"addr,omitempty"`
}

// New returns a new discovery service client
func New(machines []string) *Client {
	return &Client{
		client: etcd.NewClient(machines),
	}
}

// GetSlaves returns all the slaves registered for the users' cluster
func (c *Client) GetSlaves(username, cluster string) ([]string, error) {
	resp, err := c.client.Get(filepath.Join("/citadel/discovery", username, cluster, "slaves"), true, true)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	ips := []string{}
	for _, n := range resp.Node.Nodes {
		ips = append(ips, n.Value)
	}

	return ips, nil
}

func (c *Client) SetSlave(username, cluster, slave, addr string, ttl uint64) error {
	_, err := c.client.Set(filepath.Join("/citadel/discovery", username, cluster, "slaves", slave), addr, ttl)
	return err
}

// Delete removes the cluster and user from the discovery service
func (c *Client) Delete(username, cluster string) error {
	if _, err := c.client.Delete(filepath.Join("/citadel/discovery", username, cluster), true); err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}

		return err
	}

	return nil
}

// isNotFound returns true if the error is an etcd key not found error
func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "Key not found")
}
