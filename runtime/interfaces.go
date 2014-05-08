package runtime

type Graph interface {
	Get(string) (string, error)
	Put(string) error
}
