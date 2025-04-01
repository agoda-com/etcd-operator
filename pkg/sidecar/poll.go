package sidecar

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Poll[T client.Object](ctx context.Context, kcl client.Client, obj T, interval time.Duration, f func(obj T) (bool, error)) error {
	key := client.ObjectKeyFromObject(obj)

	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		err := kcl.Get(ctx, key, obj)
		switch {
		case apierrors.IsTooManyRequests(err):
		case err != nil:
			return err
		default:
			done, err := f(obj)
			if done {
				return err
			}
		}

		delay, ok := apierrors.SuggestsClientDelay(err)
		switch {
		case ok:
			timer.Reset(time.Duration(delay))
		default:
			timer.Reset(interval)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
}
