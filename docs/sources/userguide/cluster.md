page_title: Getting started with Docker Cluster
page_description: Introductory guide to getting a Docker cluster setup
page_keywords: documentation, docs, cluster, docker multi host, scheduling

# Getting Started with Docker Cluster


This section provides a quick introduction to getting a Docker cluster setup
on your infrastructure.

## Discovery

Before we can start deploying our container's to a Docker cluster we need to 
ensure that each of the nodes are able to discover each other.  The Hub
allows you to manage your Docker cluster easily.  To start a new cluster 
login to your Hub account and hit **Create Cluster** providing a name
of your choosing.  Under my account I will create a new cluster named
**us-east**.  After creating a cluster you should be able to see and manage 
the nodes that are currently registerd.  After creating a cluster you 
should receive a URL that looks similar to 
`https://discovery.hub.docker.com/u/crosbymichael/us-east` with your Hub 
username and cluster name.


To add a new or existing Docker
Engine to your newly created cluster use the provided URL from the Hub
with the `--discovery` flag when you start Docker in daemon mode.

    $ docker -d --discovery https://discovery.hub.docker.com/u/crosbymichael/us-east


## Master

In order to ensure consistency within the cluster one of your Docker Engines
will need to be promoted to master within the cluster.  If you are using
the Hub's discovery service you will be able to promote any of your registered
nodes to a master from the web interface.  You can also statically assign one
of the Docker Engines with the `--master` flag.

    $ docker -d --master --discovery https://discovery.hub.docker.com/u/crosbymichael/us-east


After we have one node set to the master in our cluster we will start to add additional nodes
without the `--master` flag.  These will be added as nodes within the cluster able to
accept tasks issued by the master.  For this guide we will start a three node cluster with 
each node's hostname being, **node1**, **node2**, and **node3**.  
We will promote **node1** to be the master in our cluster.


## Running your first containers

To deploy a container to the cluster you can use your existing Docker CLI to issue a
run command to any one of the nodes within the cluster.  If you issue a command to
a node in the cluster which is not the master, it will be redirected to the designated 
master.  Lets login to one of the nodes and run a redis server.

    $ docker run -d --name redis --restart always --memory 512m --cpus 0.4 -p 6379:6379 redis


In order to schedule your container to a node within the cluster that is able to run
your container you need to provide resource constraints such as cpu and memory.
These constraints are used to place your container on a machine with the resources
avaliable to fulfill your request.  If there are currently no nodes within the cluster
that are able to run your container with the requested resource constraints the container
will show up in a *PENDING* state until the request is canceled or a machine with
avaliable resources is added to the cluster.


To view the Docker Engine that is running our container we can run `docker ps` to view
that information.  

    $ docker ps
    NAMES                   IMAGE          STATUS              PORTS
    /us-east/node3/redis    redis          Up About a minute   0.0.0.0:6379->6379/tcp
 

## Failover

One of the benefits of a distributed system is that it handles failover gracefully.
If one of the nodes within your Docker cluster fails for some reason the master is
able to reschedule your containers to another machine.  Lets shutdown our **node3** 
to see how Docker handles an entire node failure.  

    $ ssh node3 && poweroff

After querying docker we should see our container go through a few states then finally
be redeployed to a new machine.

    $ docker ps
    NAMES                   IMAGE          STATUS              PORTS
    /us-east/node3/redis    redis          Exited (unknown)    0.0.0.0:6379->6379/tcp
    ...
    NAMES                   IMAGE          STATUS              PORTS
    /us-east/node3/redis    redis          Pending             0.0.0.0:6379->6379/tcp
    ...
    NAMES                   IMAGE          STATUS              PORTS
    /us-east/node1/redis    redis          Up About a minute   0.0.0.0:6379->6379/tcp


