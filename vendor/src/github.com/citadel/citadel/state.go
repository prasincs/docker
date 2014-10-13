package citadel

// State represents an entire engine's current state including all
// containers and resource information for the engine
type State struct {
	Engine     *Engine               `json:"engine,omitempty"`
	Containers map[string]*Container `json:"containers,omitempty"`
}

// ReservedCpus returns the current cpu reservation for the state
func (s *State) ReservedCpus() float64 {
	o := 0.0

	for _, c := range s.Containers {
		o += c.Image.Cpus
	}

	return o
}

// ReservedMemory returns the current memory reservation for the state
func (s *State) ReservedMemory() float64 {
	o := 0.0

	for _, c := range s.Containers {
		o += c.Image.Memory
	}

	return o
}
