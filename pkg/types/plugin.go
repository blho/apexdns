package types

type HandleFunc func(*Context)

type PluginSetupFunc func()

type Plugin interface {
	Name() string
	Handle(*Context)
}

type PluginInitializer struct {
	Name        string
	Description string
	SetupFunc   PluginSetupFunc
}

type ZoneConfig struct {
	Zone string
}
