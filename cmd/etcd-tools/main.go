package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/agoda-com/etcd-operator/cmd/etcd-tools/cmd"
)

func main() {
	ctx := signals.SetupSignalHandler()
	root := cmd.RootCommand()
	err := root.ExecuteContext(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, root.UsageString())

		os.Exit(1)
	}
}
