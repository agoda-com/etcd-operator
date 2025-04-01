package cmd

import (
	"flag"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func RootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "etcd-tools",
		SilenceErrors: true,
	}

	zapOptions := &zap.Options{}
	gflags := &flag.FlagSet{}
	zapOptions.BindFlags(gflags)
	cmd.Flags().AddGoFlagSet(gflags)

	// setup context with global logger
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logger := zap.New(zap.UseFlagOptions(zapOptions))
		log.SetLogger(logger)
		cmd.SetContext(log.IntoContext(cmd.Context(), logger))
	}

	cmd.AddCommand(BackupCommand())
	cmd.AddCommand(DefragCommand())
	cmd.AddCommand(RestoreCommand())

	return cmd
}
