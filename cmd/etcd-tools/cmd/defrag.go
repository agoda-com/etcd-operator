package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv3 "go.etcd.io/etcd/client/v3"

	"k8s.io/apimachinery/pkg/api/resource"
)

func DefragCommand() *cobra.Command {
	cmd := &cobra.Command{
		Short: "Defragment cluster members",
		Use:   "defrag [--credentials-dir DIR] [--endpoint ENDPOINT] [--unused-ratio RATIO] [--unused-size SIZE]",
	}

	flags := cmd.Flags()
	endpoint := flags.String("endpoint", "", "etcd endpoint")
	credentialsDir := flags.String("credentials-dir", "", "etcd credentials directory")

	ratio := flags.Float64("unused-ratio", 0.7, "threshold ratio of unused space")
	size := resource.QuantityValue{
		Quantity: resource.MustParse("128M"),
	}
	flags.Var(&size, "unused-size", "threshold size of unused space")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		tlsConfig, err := etcd.TLSConfig(etcd.LoadDir(os.DirFS(*credentialsDir)))
		if err != nil {
			return err
		}

		ecl, err := etcd.Connect(ctx, tlsConfig, *endpoint)
		if err != nil {
			return fmt.Errorf("connect etcd: %w", err)
		}

		return Defrag(ctx, ecl, DefragParams{
			Ratio: *ratio,
			Size:  size.Value(),
		})
	}

	return cmd
}

type DefragParams struct {
	Ratio float64
	Size  int64
}

func Defrag(ctx context.Context, ecl *etcdv3.Client, params DefragParams) error {
	logger := log.FromContext(ctx)

	members, err := ecl.MemberList(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, member := range members.Members {
		if member.IsLearner || len(member.ClientURLs) == 0 {
			continue
		}

		endpoint := member.ClientURLs[0]
		logger := logger.WithValues("endpoint", endpoint)

		memberStatus, err := ecl.Status(ctx, endpoint)
		if err != nil {
			errs = append(errs, fmt.Errorf("status: %w", err))
			continue
		}

		unused := memberStatus.DbSize - memberStatus.DbSizeInUse
		ratio := float64(memberStatus.DbSize) / float64(memberStatus.DbSizeInUse)
		if ratio > params.Ratio && unused > params.Size {
			logger.Info("skipped")
			continue
		}

		_, err = ecl.Compact(ctx, memberStatus.Header.Revision)
		if err != nil {
			errs = append(errs, fmt.Errorf("compact: %w", err))
			continue
		}

		_, err = ecl.Defragment(ctx, endpoint)
		if err != nil {
			errs = append(errs, fmt.Errorf("defragment: %w", err))
			continue
		}

		logger.Info("defragmented")
	}

	return errors.Join(errs...)
}
