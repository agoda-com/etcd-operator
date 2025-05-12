package etcd

import (
	"context"
	"crypto/tls"
	"sync"

	simplelru "github.com/hashicorp/golang-lru/v2/simplelru"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TLSCache struct {
	kcl client.Client

	mtx  sync.Mutex
	data *simplelru.LRU[client.ObjectKey, *cacheItem]
}

func NewTLSCache(kcl client.Client, size int) (*TLSCache, error) {
	data, err := simplelru.NewLRU[client.ObjectKey, *cacheItem](size, nil)
	if err != nil {
		return nil, err
	}

	return &TLSCache{
		kcl:  kcl,
		data: data,
	}, nil
}

func (c *TLSCache) Get(ctx context.Context, key client.ObjectKey) (*tls.Config, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// get partial metadata to retrieve resource version
	partial := metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: key.Namespace,
			Name:      key.Name,
		},
	}
	err := c.kcl.Get(ctx, key, &partial)
	if err != nil {
		return nil, err
	}

	// singleflight loading tls config from secret
	item := c.cacheItem(key, partial.ResourceVersion)
	item.once.Do(func() {
		item.done = make(chan struct{}, 1)
		defer close(item.done)

		item.config, item.err = TLSConfig(LoadSecret(ctx, c.kcl, key))
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-item.done:
		return item.config, item.err
	}
}

func (c *TLSCache) cacheItem(key client.ObjectKey, version string) *cacheItem {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	item, ok := c.data.Get(key)
	if !ok || item.version != version {
		item = &cacheItem{
			version: version,
		}
		c.data.Add(key, item)
	}

	return item
}

type cacheItem struct {
	once sync.Once
	done chan struct{}

	version string
	config  *tls.Config
	err     error
}
