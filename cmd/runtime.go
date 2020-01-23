package main

import (
	"os"
	"runtime"

	"github.com/oif/gokit/runtime/dumpstack"
)

func setupRuntime() {
	dumpstack.SetupTrap()

	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}
