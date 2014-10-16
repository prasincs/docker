package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/citadel/citadel"
	"github.com/citadel/citadel/cluster"
	"github.com/docker/docker/pkg/log"
	"github.com/gorilla/mux"
	"github.com/samalba/dockerclient"
)

func init() {
	EventsHandler = &eventsHandler{
		ws: make(map[string]io.Writer),
		cs: make(map[string]chan struct{}),
	}
}

type HttpApiFunc func(c *cluster.Cluster, w http.ResponseWriter, r *http.Request)

func ping(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte{'O', 'K'})
}

func postContainersCreate(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var image citadel.Image
	var config dockerclient.ContainerConfig

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		fmt.Println("Create Error:", err)
	}

	var ports []*citadel.Port
	for port, bindings := range config.HostConfig.PortBindings {
		parts := strings.SplitN(port, "/", 2)
		p, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		for _, binding := range bindings {
			hp, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				continue
			}
			port := citadel.Port{
				Proto:         parts[1],
				HostIp:        binding.HostIp,
				Port:          hp,
				ContainerPort: p,
			}
			ports = append(ports, &port)
		}
	}

	for port, _ := range config.ExposedPorts {
		image.ExposedPorts = append(image.ExposedPorts, port)
	}

	image.Publish = config.HostConfig.PublishAllPorts
	image.BindPorts = ports
	image.Name = config.Image
	image.Args = config.Cmd
	image.Type = "service"
	image.Memory = float64(config.Memory) / 1024 / 1024
	image.Cpus = float64(config.CpuShares)
	image.ContainerName = r.Form.Get("name")

	image.Environment = make(map[string]string)
	image.Labels = make(map[string]string)
	// Fill out env and labels.
	for _, e := range config.Env {
		switch {
		case strings.HasPrefix(e, "constraint:"):
			constraint := strings.TrimPrefix(e, "constraint:")
			parts := strings.SplitN(constraint, "=", 2)
			image.Labels[parts[0]] = parts[1]
		default:
			parts := strings.SplitN(e, "=", 2)
			image.Environment[parts[0]] = parts[1]
		}
	}

	container, err := c.Create(&image, false)
	if err == nil {
		fmt.Fprintf(w, "{%q:%q}", "Id", container.ID)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func postContainersStart(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	container := c.FindContainer(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}

	if err := c.Start(container); err == nil {
		fmt.Fprintf(w, "{%q:%q}", "Id", container.ID)
	} else {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func deleteContainers(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := mux.Vars(r)["name"]
	force := r.Form.Get("force") == "1"
	container := c.FindContainer(name)
	if container == nil {
		http.Error(w, fmt.Sprintf("Container %s not found", name), http.StatusNotFound)
		return
	}
	if err := c.Remove(container, force); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func redirectContainer(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	container := c.FindContainer(mux.Vars(r)["name"])
	if container != nil {
		newURL, _ := url.Parse(container.Engine.Addr)
		newURL.RawQuery = r.URL.RawQuery
		newURL.Path = r.URL.Path
		fmt.Println("REDIR ->", newURL.String())
		http.Redirect(w, r, newURL.String(), http.StatusSeeOther)
	}
}

// FIXME: this is ugly
func getContainerJSON(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	container := c.FindContainer(mux.Vars(r)["name"])
	if container != nil {
		resp, err := http.Get(container.Engine.Addr + "/containers/" + container.ID + "/json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(bytes.Replace(data, []byte("\"HostIp\":\"0.0.0.0\""), []byte(fmt.Sprintf("\"HostIp\":%q", container.Engine.IP)), -1))
	}
}

func getContainersJSON(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	var (
		err              error
		containers       []*citadel.Container
		dockerContainers = []dockerclient.Container{}
	)

	// Options parsing.
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	all := r.Form.Get("all") == "1"

	if containers, err = c.ListContainers(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, cs := range containers {
		// Skip stopped containers unless -a was specified.
		if cs.State != "running" && cs.State != "pending" && !all {
			continue
		}
		dockerContainers = append(dockerContainers, citadel.ToDockerContainer(cs))
	}

	sort.Sort(sort.Reverse(ContainerSorter(dockerContainers)))
	json.NewEncoder(w).Encode(dockerContainers)
}

func getInfo(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	containers, err := c.ListContainers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var driverStatus [][2]string

	for _, engine := range c.Engines() {
		driverStatus = append(driverStatus, [2]string{engine.ID, engine.Addr})
	}
	info := struct {
		Containers                             int
		Driver, ExecutionDriver                string
		DriverStatus                           [][2]string
		KernelVersion, OperatingSystem         string
		MemoryLimit, SwapLimit, IPv4Forwarding bool
	}{
		len(containers),
		"libcluster", "libcluster",
		driverStatus,
		"N/A", "N/A",
		true, true, true,
	}

	json.NewEncoder(w).Encode(info)
}

type eventsHandler struct {
	sync.RWMutex
	ws map[string]io.Writer
	cs map[string]chan struct{}
}

var EventsHandler *eventsHandler

func (eh *eventsHandler) Handle(e *citadel.Event) error {
	eh.RLock()
	str := fmt.Sprintf("{%q:%q,%q:%q,%q:%q,%q:%d}", "status", e.Type, "id", e.Container.ID, "from", e.Container.Image.Name+" node:"+e.Container.Engine.ID, "time", e.Time.Unix())

	for key, w := range eh.ws {
		if _, err := fmt.Fprintf(w, str); err != nil {
			close(eh.cs[key])
			delete(eh.ws, key)
			delete(eh.cs, key)
			continue
		}

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

	}
	eh.RUnlock()
	return nil
}

func getEvents(c *cluster.Cluster, w http.ResponseWriter, r *http.Request) {
	EventsHandler.Lock()
	EventsHandler.ws[r.RemoteAddr] = w
	EventsHandler.cs[r.RemoteAddr] = make(chan struct{})
	EventsHandler.Unlock()
	w.Header().Set("Content-Type", "application/json")

	<-EventsHandler.cs[r.RemoteAddr]
}

func createRouter(c *cluster.Cluster) (*mux.Router, error) {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/_ping":  ping,
			"/events": getEvents,
			"/info":   getInfo,
			//#			"/version": getVersion,
			//			"/images/json":                    getImagesJSON,
			//			"/images/viz":                     getImagesViz,
			//			"/images/search":                  getImagesSearch,
			//			"/images/get":                     getImagesGet,
			//			"/images/{name:.*}/get":           getImagesGet,
			//			"/images/{name:.*}/history":       getImagesHistory,
			//			"/images/{name:.*}/json":          getImagesByName,
			"/containers/ps":                getContainersJSON,
			"/containers/json":              getContainersJSON,
			"/containers/{name:.*}/export":  redirectContainer,
			"/containers/{name:.*}/changes": redirectContainer,
			"/containers/{name:.*}/json":    getContainerJSON,
			"/containers/{name:.*}/top":     redirectContainer,
			"/containers/{name:.*}/logs":    redirectContainer,
			//			"/containers/{name:.*}/attach/ws": wsContainersAttach,
		},
		"POST": {
			//			"/auth":                         postAuth,
			//			"/commit":                       postCommit,
			//			"/build":                        postBuild,
			//			"/images/create":                postImagesCreate,
			//			"/images/load":                  postImagesLoad,
			//			"/images/{name:.*}/push":        postImagesPush,
			//			"/images/{name:.*}/tag":         postImagesTag,
			"/containers/create": postContainersCreate,
			//# "/containers/{name:.*}/kill": postContainersKill,
			//#			"/containers/{name:.*}/pause":   postContainersPause,
			//#			"/containers/{name:.*}/unpause": postContainersUnpause,
			//#"/containers/{name:.*}/restart": postContainersRestart,
			"/containers/{name:.*}/start": postContainersStart,
			//#"/containers/{name:.*}/stop":    postContainersStop,
			//			"/containers/{name:.*}/wait":    postContainersWait,
			//			"/containers/{name:.*}/resize":  postContainersResize,
			//			#"/containers/{name:.*}/attach": postContainersAttach,
			//			"/containers/{name:.*}/copy":    postContainersCopy,
			//			"/containers/{name:.*}/exec":    postContainerExecCreate,
			//			"/exec/{name:.*}/start":         postContainerExecStart,
			//			"/exec/{name:.*}/resize":        postContainerExecResize,
		},
		"DELETE": {
			"/containers/{name:.*}": deleteContainers,
		},
		//			"/images/{name:.*}":     deleteImages,
		//#		},
		//		"OPTIONS": {
		//			"": optionsHandler,
		//		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			log.Debugf("Registering %s, %s", method, route)
			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, r *http.Request) {
				fmt.Printf("-> %s %s\n", r.Method, r.RequestURI)
				localFct(c, w, r)
			}
			localMethod := method

			// add the new route
			r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	return r, nil
}

func ListenAndServe(c *cluster.Cluster, addr string) error {
	r, err := createRouter(c)
	if err != nil {
		return err
	}
	s := &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s.ListenAndServe()
}
