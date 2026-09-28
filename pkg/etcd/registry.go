package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/cloudwego/kitex/pkg/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const registryPrefix = "kitex/registry-etcd"

type instanceInfo struct {
	Network string            `json:"network"`
	Address string            `json:"address"`
	Weight  int               `json:"weight"`
	Tags    map[string]string `json:"tags"`
}

type etcdRegistry struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
	cancel  context.CancelFunc
}

func NewEtcdRegistry(
	endpoints []string,
) (registry.Registry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &etcdRegistry{
		client: client,
	}, nil
}

func (e *etcdRegistry) Register(info *registry.Info) error {
	if info == nil || info.ServiceName == "" || info.Addr == nil {
		return errors.New("invalid registry information")
	}

	leaseContext, leaseCancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer leaseCancel()

	lease, err := e.client.Grant(leaseContext, 60)
	if err != nil {
		return err
	}

	value, err := json.Marshal(instanceInfo{
		Network: info.Addr.Network(),
		Address: info.Addr.String(),
		Weight:  info.Weight,
		Tags:    info.Tags,
	})
	if err != nil {
		return err
	}

	key := registryKey(info.ServiceName, info.Addr.String())

	putContext, putCancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer putCancel()

	_, err = e.client.Put(
		putContext,
		key,
		string(value),
		clientv3.WithLease(lease.ID),
	)
	if err != nil {
		return err
	}

	keepAliveContext, cancel := context.WithCancel(
		context.Background(),
	)

	keepAliveChannel, err := e.client.KeepAlive(
		keepAliveContext,
		lease.ID,
	)
	if err != nil {
		cancel()
		return err
	}

	e.leaseID = lease.ID
	e.cancel = cancel

	go func() {
		for range keepAliveChannel {
			// 持续读取响应，保持租约有效
		}
	}()

	return nil
}

func (e *etcdRegistry) Deregister(info *registry.Info) error {
	if info == nil || info.ServiceName == "" || info.Addr == nil {
		return errors.New("invalid registry information")
	}

	key := registryKey(info.ServiceName, info.Addr.String())

	deleteContext, deleteCancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer deleteCancel()

	_, err := e.client.Delete(deleteContext, key)

	if e.cancel != nil {
		e.cancel()
	}

	return err
}

func registryKey(serviceName string, address string) string {
	return registryPrefix + "/" + serviceName + "/" + address
}
