package types

type HandleFunc func(*Context)

type Plugin interface {
	Name() string
	Handle(*Context)
}
