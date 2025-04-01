package main

import (
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	cmd := Command()
	err := cmd.ExecuteContext(signals.SetupSignalHandler())
	if err != nil {
		os.Exit(1)
	}
}
