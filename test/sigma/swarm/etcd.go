package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/util/json"
)

var (
	// etcdclient used to connect etcd server.
	etcdClient *clientv3.Client
	// Timeout of connecting to etcd (default: 10s)
	Timeout = time.Second * 10
)

// GetEtcdClient initializes a new etcd client of a existed etcd cluster or create a new etcd cluster.
func GetEtcdClient() (*clientv3.Client, error) {
	var err error

	if etcdClient != nil {
		return etcdClient, nil
	}

	if EtcdEndpoints == nil {
		return nil, fmt.Errorf("EtcdEndpoints is nil")
	}

	etcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   EtcdEndpoints,
		DialTimeout: Timeout,
	})

	return etcdClient, err
}

// Close close etcdClient.
func Close() error {
	return etcdClient.Close()
}

func handleErr(err error) error {
	switch err {
	case context.Canceled:
		return fmt.Errorf("ctx is canceled by another routine: %v", err)
	case context.DeadlineExceeded:
		return fmt.Errorf("ctx is attached with a deadline is exceeded: %v", err)
	case rpctypes.ErrEmptyKey:
		return fmt.Errorf("client-side error: %v", err)
	default:
		return fmt.Errorf("bad cluster endpoints, which are not etcd servers: %v", err)
	}
}

func etcdPut(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return err
	}
	_, err = etcdclient.Put(ctx, key, value)
	cancel()
	if err != nil {
		return handleErr(err)
	}

	return nil
}

// EtcdGet get one key's value.
func EtcdGet(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return nil, err
	}

	res, err := etcdclient.Get(ctx, key)
	cancel()
	if err != nil {
		return []byte{}, handleErr(err)
	}

	if len(res.Kvs) == 0 {
		return nil, nil
	}

	if len(res.Kvs) != 1 {
		return []byte{}, errors.Errorf("Failed get key[%s],not one key, EtcdEndpoints:%s", key, EtcdEndpoints)
	}

	return res.Kvs[0].Value, nil
}

func EtcdDelete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return err
	}
	etcdclient.Delete(ctx, key)
	if err != nil {
		return handleErr(err)
	}
	return nil
}

func EtcdPut(key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return err
	}
	bytes, _ := json.Marshal(value)
	_, err = etcdclient.Put(ctx, key, string(bytes))
	if err != nil {
		return handleErr(err)
	}
	return nil
}

func EtcdPutString(key,value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return err
	}
	_, err = etcdclient.Put(ctx, key, value)
	if err != nil {
		return handleErr(err)
	}
	return nil
}

// EtcdGetPrefix returns the number of key which has specific prefix.
func EtcdGetPrefix(prefixKey string) ([]*mvccpb.KeyValue, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return nil, err
	}
	res, err := etcdclient.Get(ctx, prefixKey, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, handleErr(err)
	}
	return res.Kvs, nil
}

// etcdDeletePrefix delete key which has specific prefix.
func etcdDeletePrefix(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	etcdclient, err := GetEtcdClient()
	if err != nil {
		return err
	}
	_, err = etcdclient.Delete(ctx, key, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return handleErr(err)
	}

	logrus.Infof("EtcdClient delete prefix key[%s] successfully", key)
	return nil
}
