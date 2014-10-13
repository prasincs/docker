package main

import (
	"log"

	"github.com/citadel/citadel/docker"
)

func main() {
	// temporary register 3 nodes

	// Ubuntu boxes
	if err := docker.Slave("http://discovery.crosbymichael.com/u/citadel_test/cluster", "node1", "http://ec2-54-68-133-155.us-west-2.compute.amazonaws.com:4242"); err != nil {
		log.Fatal(err)
	}
	if err := docker.Slave("http://discovery.crosbymichael.com/u/citadel_test/cluster", "node2", "http://ec2-54-69-225-30.us-west-2.compute.amazonaws.com:4242"); err != nil {
		log.Fatal(err)
	}
	// Fedora
	if err := docker.Slave("http://discovery.crosbymichael.com/u/citadel_test/cluster", "node3", "http://ec2-54-69-11-29.us-west-2.compute.amazonaws.com:4242"); err != nil {
		log.Fatal(err)
	}

	log.Fatal(docker.Master("http://discovery.crosbymichael.com/u/citadel_test/cluster", ":4243"))
}
