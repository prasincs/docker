package main

import (
	"fmt"
	"net/http"

	"github.com/citadel/citadel/discovery"
	"github.com/gorilla/mux"
)

type requestInfo struct {
	Username string
	Cluster  string
	SlaveID  string
}

func newRequestInfo(r *http.Request) requestInfo {
	vars := mux.Vars(r)

	return requestInfo{
		Username: vars["username"],
		Cluster:  vars["cluster"],
		SlaveID:  vars["slave"],
	}
}

func writeError(w http.ResponseWriter, err error, info requestInfo) {
	if err == discovery.ErrNotFound {
		http.Error(w, fmt.Sprintf("cluster for %s/%s not found", info.Username, info.Cluster), http.StatusNotFound)
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
