package discovery

import (
	"common/tlog"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

//DependType 服务依赖类型
type DependType int

const (
	//DependNormal 非强依赖, 不会堵塞
	DependNormal = DependType(0)
	//DependMust 强依赖, 订阅一刻就要确立连接
	DependMust = DependType(1)
	//DependBlock 强依赖, 会堵塞等待确立连接
	DependBlock = DependType(2)

	_ResolverSchema = "discovery"
	_ResolverFormat = "/%s/%s/%s/"
	_ResolverTarget = "%s://%s/%s"
)

//Resolver ...
func Resolver(env, name string, typ DependType) (*grpc.ClientConn, ConsistSplitter) {
	tlog.Infof("Resolver: env=%s services=%s", env, name)
	if etcdClient == nil {
		panic("Subscribe: Please Init Ected Connection Firstly")
	}
	if env == "" {
		panic("Subscribe: Error=env is empty")
	}

	rn := &ResolverNode{dependType: typ}
	resolver.Register(resolver.Builder(rn))

	//注册自定义的 balancer
	b := &balancerDiscovery{name: name}
	balancer.Register(b)

	target := fmt.Sprintf(_ResolverTarget, rn.Scheme(), env, name)
	dailOpts := []grpc.DialOption{
		grpc.WithBalancerName(name),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024 * 1024 * 16),
		),
	}
	if typ == DependBlock {
		dailOpts = append(dailOpts, grpc.WithBlock())
	}
	conn, err := grpc.Dial(target, dailOpts...)
	if err != nil {
		panic(err)
	}
	return conn, b
}

//Builder.Build ...
func (this *ResolverNode) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	this.dir = fmt.Sprintf(_ResolverFormat, target.Scheme, target.Authority, target.Endpoint)
	this.services = []*Service{}
	this.resolverConn = cc

	subscribes[target.Endpoint] = this

	this.subscribe(this.dependType != DependNormal) //初始订阅依赖
	go this.watching()                              //监控订阅变化

	return this, nil
}

//Builder.Scheme ....
func (ns *ResolverNode) Scheme() string {
	return _ResolverSchema
}

//Resolver.ResolveNow ...
func (ns *ResolverNode) ResolveNow(rn resolver.ResolveNowOptions) {
	//ignore
}

//Resolver.Close ...
func (ns *ResolverNode) Close() {
	//ignore
}
