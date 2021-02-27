package discovery

import (
	"common/tlog"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/resolver"
)

//ErrTargetOver ...
var ErrTargetOver = errors.New("Target Over")

type ConsistSplitter interface {
	ConsistSplit(ids []int64) map[int][]int64
}

type balancerDiscovery struct {
	sync.Mutex
	name string
	cc   balancer.ClientConn

	csEvltr *connectivityStateEvaluator
	state   connectivity.State

	subConns map[resolver.Address]balancer.SubConn
	scStates map[balancer.SubConn]connectivity.State
	picker   *picker
}

type picker struct {
	name      string
	err       error
	consist   *Consistent
	scAddrMap map[string]balancer.SubConn
	scList    []balancer.SubConn
	next      int64
}

// Name returns the name of balancers built by this builder.
// It will be used to pick balancers (for example in service config).
// Builder.Name
func (b *balancerDiscovery) Name() string {
	return b.name
}

// Build creates a new balancer with the ClientConn.
// Builder.Build
func (b *balancerDiscovery) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	b.cc = cc
	b.subConns = make(map[resolver.Address]balancer.SubConn)
	b.scStates = make(map[balancer.SubConn]connectivity.State)
	b.csEvltr = &connectivityStateEvaluator{}
	return balancer.Balancer(b)
}

// Balancer.HandleResolvedAddrs 每当可用服务地址变更, gRPC 都会交付一组地址给该方法
// 这里可以选择创建/移除一些地址.
// 如果 Resolver 给了一组空地址, 或者 err!=nil 则 return
func (b *balancerDiscovery) HandleResolvedAddrs(addrs []resolver.Address, err error) {
	if err != nil {
		tlog.Debugf("HandleResolvedAddrs called with error %v", err)
		return
	}
	tlog.Debugf("got new resolved addresses: %v", addrs)

	b.Lock()
	defer b.Unlock()
	addrsSet := make(map[resolver.Address]struct{})
	for _, a := range addrs {
		addrsSet[a] = struct{}{}
		if _, ok := b.subConns[a]; !ok {
			// a is a new address (not existing in b.subConns).
			sc, err := b.cc.NewSubConn([]resolver.Address{a}, balancer.NewSubConnOptions{})
			if err != nil {
				tlog.Warningf("failed to create new SubConn: %v", err)
				continue
			}
			b.subConns[a] = sc
			b.scStates[sc] = connectivity.Idle
			sc.Connect()
		}
	}
	for a, sc := range b.subConns {
		// a was removed by resolver.
		if _, ok := addrsSet[a]; !ok {
			tlog.Debug("remove one subconn", a)
			b.cc.RemoveSubConn(sc)
			delete(b.subConns, a)
			// Keep the state of this sc in b.scStates until sc's state becomes Shutdown.
			// The entry will be deleted in HandleSubConnStateChange.
		}
	}
}

// Balancer.HandleSubConnStateChange
func (b *balancerDiscovery) HandleSubConnStateChange(sc balancer.SubConn, s connectivity.State) {
	tlog.Debugf("handle SubConn state change: %p, %v", sc, s)
	b.Lock()
	defer b.Unlock()
	oldS, ok := b.scStates[sc]
	if !ok {
		tlog.Debugf("got state changes for an unknown SubConn: %p, %v", sc, s)
		return
	}
	b.scStates[sc] = s
	switch s {
	case connectivity.Idle:
		sc.Connect()
	case connectivity.Shutdown:
		// When an address was removed by resolver, b called RemoveSubConn but
		// kept the sc's state in scStates. Remove state for this sc here.
		delete(b.scStates, sc)
	}

	oldAggrState := b.state
	b.state = b.csEvltr.recordTransition(oldS, s)

	// Regenerate picker when one of the following happens:
	//  - this sc became ready from not-ready
	//  - this sc became not-ready from ready
	//  - the aggregated state of balancer became TransientFailure from non-TransientFailure
	//  - the aggregated state of balancer became non-TransientFailure from TransientFailure

	tlog.Debugf("HandleSubConnStateChange state s [%s] oldS [%s] b.state[%s] oldAggrState[%s]", s, oldS, b.state, oldAggrState)
	if b.picker == nil ||
		(s == connectivity.Ready) != (oldS == connectivity.Ready) ||
		(b.state == connectivity.TransientFailure) != (oldAggrState == connectivity.TransientFailure) {
		b.regeneratePicker()
	}

	b.cc.UpdateState(balancer.State{ConnectivityState: b.state, Picker: b.picker})
	return
}

// Balancer.Close
func (b *balancerDiscovery) Close() {
}

// regeneratePicker takes a snapshot of the balancer, and generates a picker
// from it. The picker
//  - always returns ErrTransientFailure if the balancer is in TransientFailure,
//  - or does round robin selection of all READY SubConns otherwise.
func (b *balancerDiscovery) regeneratePicker() {
	if b.state == connectivity.TransientFailure {
		b.picker = b.newPicker(nil, balancer.ErrTransientFailure)
		return
	}
	b.picker = b.newPicker(b.subConns, nil)
}

func (b *balancerDiscovery) newPicker(subcMap map[resolver.Address]balancer.SubConn, err error) *picker {
	tlog.Debugf("newPicker called with scs: %v, %v", subcMap, err)
	if err != nil {
		return &picker{err: err}
	}
	consist := NewConsistent()
	scAddrMap := map[string]balancer.SubConn{}
	scList := []balancer.SubConn{}

	for addr, sc := range subcMap {
		// TransientFailure indicates the ClientConn has seen a failure but expects to recover.
		// Shutdown indicates the ClientConn has started shutting down.
		stat := b.scStates[sc]
		if stat != connectivity.Shutdown && stat != connectivity.TransientFailure {
			consist.Add(addr.Addr)
			scAddrMap[addr.Addr] = sc
			scList = append(scList, sc)
		}
	}

	return &picker{
		name:      b.name,
		scList:    scList,
		scAddrMap: scAddrMap,
		consist:   consist,
	}
}

// ConsistSplit 返回一组 ids 对应的 target 机器
func (b *balancerDiscovery) ConsistSplit(ids []int64) map[int][]int64 {
	var m = map[int][]int64{}
	if b.picker != nil {
		p := b.picker
		for _, id := range ids {
			if hit, err := p.consist.Get(strconv.FormatInt(id, 10)); err == nil {
				scTarget := p.scAddrMap[hit]
				target := 0
				for i, sc := range p.scList {
					if sc == scTarget {
						target = i + 1
						break
					}
				}
				if target > 0 {
					if len(m[target]) == 0 {
						m[target] = []int64{id}
					} else {
						m[target] = append(m[target], id)
					}
				}
			}
		}
	}
	return m
}

func (p *picker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	var res balancer.PickResult

	if p.err != nil {
		return res, p.err
	}

	slen := len(p.scList)
	if slen <= 0 {
		return res, balancer.ErrNoSubConnAvailable
	}

	//广播; 调用者会依次递增 target, 直到收到 ErrTargetOver 错误后 Break 调用循环
	target, _ := info.Ctx.Value(CtxKey("target")).(int)
	if target > 0 {
		if target <= slen {
			tlog.Debugf("Target Routing: target=%d", target)
			res.SubConn = p.scList[target-1]
			return res, nil
		}
		tlog.Debug("Target Routing: Over")
		return res, ErrTargetOver
	}

	//一致性 hash
	routing, _ := info.Ctx.Value(CtxKey("routing")).(string)
	if routing != "" {
		hit, err := p.consist.Get(routing)
		if err == nil {
			tlog.Debugf("Consist Routing %s To %s", routing, hit)
			res.SubConn = p.scAddrMap[hit]
			return res, nil
		}
		tlog.Debugf("Consist Routing Error=%s, routing=%s", err.Error(), routing)
	}

	//轮询
	next := atomic.AddInt64(&p.next, 1)
	next = next % int64(slen)
	tlog.Debugf("Round Routing To %s/%d", p.name, next)
	res.SubConn = p.scList[next]
	return res, nil
}

// connectivityStateEvaluator gets updated by addrConns when their
// states transition, based on which it evaluates the state of
// ClientConn.
type connectivityStateEvaluator struct {
	numReady            uint64 // Number of addrConns in ready state.
	numConnecting       uint64 // Number of addrConns in connecting state.
	numTransientFailure uint64 // Number of addrConns in transientFailure.
}

// recordTransition records state change happening in every subConn and based on
// that it evaluates what aggregated state should be.
// It can only transition between Ready, Connecting and TransientFailure. Other states,
// Idle and Shutdown are transitioned into by ClientConn; in the beginning of the connection
// before any subConn is created ClientConn is in idle state. In the end when ClientConn
// closes it is in Shutdown state.
//
// recordTransition should only be called synchronously from the same goroutine.
func (cse *connectivityStateEvaluator) recordTransition(oldState, newState connectivity.State) connectivity.State {
	// Update counters.
	for idx, state := range []connectivity.State{oldState, newState} {
		updateVal := 2*uint64(idx) - 1 // -1 for oldState and +1 for new.
		switch state {
		case connectivity.Ready:
			cse.numReady += updateVal
		case connectivity.Connecting:
			cse.numConnecting += updateVal
		case connectivity.TransientFailure:
			cse.numTransientFailure += updateVal
		}
	}

	// Evaluate.
	if cse.numReady > 0 {
		return connectivity.Ready
	}
	if cse.numConnecting > 0 {
		return connectivity.Connecting
	}
	return connectivity.TransientFailure
}
