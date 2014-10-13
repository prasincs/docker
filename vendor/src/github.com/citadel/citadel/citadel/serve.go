package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/citadel/citadel/discovery"
	"github.com/codegangsta/cli"
	"github.com/gorilla/mux"
)

var serveCommand = cli.Command{
	Name:  "discovery",
	Usage: "serve the REST api for the discovery service",
	Flags: []cli.Flag{
		cli.StringFlag{Name: "addr", Value: ":8080", Usage: "ip and port to serve the HTTP api"},
		cli.IntFlag{Name: "ttl", Value: 30, Usage: "set the default ttl that is required for slave information"},
	},
	Action: serveAction,
}

type server struct {
	r      *mux.Router
	client *discovery.Client
	ttl    uint64
}

func newServer(context *cli.Context) http.Handler {
	s := &server{
		r:      mux.NewRouter(),
		client: discovery.New(context.GlobalStringSlice("etcd")),
		ttl:    uint64(context.Int("ttl")),
	}

	// list the slaves in the cluster
	s.r.HandleFunc("/u/{username:.*}/{cluster:.*}", s.listClusterSlaves).Methods("GET")

	// update slave information for the cluster
	s.r.HandleFunc("/u/{username:.*}/{cluster:.*}/{slave:.*}", s.updateSlave).Methods("POST")

	// delete the cluster
	s.r.HandleFunc("/u/{username:.*}/{cluster:.*}", s.deleteCluster).Methods("DELETE")

	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

// listClusterSlaves returns a list of all the slave's addresses in the cluster
// for the specific user and cluster name
//
// GET /u/crosbymichael/testcluster
//
// ["192.168.56.1:2375"]
func (s *server) listClusterSlaves(w http.ResponseWriter, r *http.Request) {
	info := newRequestInfo(r)

	ips, err := s.client.GetSlaves(info.Username, info.Cluster)
	if err != nil {
		logger.WithField("error", err).Error("get slaves for cluster")

		writeError(w, err, info)

		return
	}

	w.Header().Set("content-type", "application/json")
	if len(ips) == 0 {
		if _, err := w.Write([]byte("[]")); err != nil {
			logger.WithField("error", err).Error("encode slave ips")
		}

		return
	}

	if err := json.NewEncoder(w).Encode(ips); err != nil {
		logger.WithField("error", err).Error("encode slave ips")
	}
}

func (s *server) updateSlave(w http.ResponseWriter, r *http.Request) {
	info := newRequestInfo(r)

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.WithField("error", err).Error("read request body for addr")

		writeError(w, err, info)
		return
	}

	if err := s.client.SetSlave(info.Username, info.Cluster, info.SlaveID, string(data), s.ttl); err != nil {
		logger.WithField("error", err).Error("read request body for addr")

		writeError(w, err, info)
		return
	}
}

func (s *server) deleteCluster(w http.ResponseWriter, r *http.Request) {
	info := newRequestInfo(r)

	if err := s.client.Delete(info.Username, info.Cluster); err != nil {
		logger.WithField("error", err).Error("list cluster slaves")

		writeError(w, err, info)
		return
	}
}

func serveAction(context *cli.Context) {
	s := newServer(context)

	logger.WithField("addr", context.String("addr")).Info("start discovery service")

	if err := http.ListenAndServe(context.String("addr"), s); err != nil {
		logger.Fatal(err)
	}
}
