package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/kitex/pkg/discovery"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const defaultWeight = 10

type etcdResolver struct {
	client *clientv3.Client
}

func NewEtcdResolver(
	endpoints []string,
) (discovery.Resolver, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &etcdResolver{
		client: client,
	}, nil
}

func (e *etcdResolver) Target(
	ctx context.Context,
	target rpcinfo.EndpointInfo,
) string {
	return target.ServiceName()
}

func (e *etcdResolver) Resolve(
	ctx context.Context,
	serviceName string,
) (discovery.Result, error) {
	prefix := registryPrefix + "/" + serviceName

	response, err := e.client.Get(
		ctx,
		prefix,
		clientv3.WithPrefix(),
	)
	if err != nil {
		return discovery.Result{}, err
	}

	instances := make([]discovery.Instance, 0, len(response.Kvs))

	for _, item := range response.Kvs {
		var info instanceInfo

		if err := json.Unmarshal(item.Value, &info); err != nil {
			continue
		}

		weight := info.Weight
		if weight <= 0 {
			weight = defaultWeight
		}

		instances = append(
			instances,
			discovery.NewInstance(
				info.Network,
				info.Address,
				weight,
				info.Tags,
			),
		)
	}

	if len(instances) == 0 {
		return discovery.Result{}, fmt.Errorf(
			"no instance found for service %s",
			serviceName,
		)
	}

	return discovery.Result{
		Cacheable: true,
		CacheKey:  serviceName,
		Instances: instances,
	}, nil
}

func (e *etcdResolver) Diff(
	cacheKey string,
	previous discovery.Result,
	next discovery.Result,
) (discovery.Change, bool) {
	return discovery.DefaultDiff(cacheKey, previous, next)
}

func (e *etcdResolver) Name() string {
	return "etcd"
}