package discovery

//目前仅支持 gRPC 服务的注册

import (
	"common/tlog"
	"common/util"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc/resolver"

	etcd "go.etcd.io/etcd/clientv3"
)

const (
	//服务发现的 Key Prefix: /discovery/{env}/{service}
	_DirectoryFormat = "/discovery/%s/%s/"

	//除了 Watching 之外, 目录十秒钟再收录一次, 避免遗漏
	_DirectoryInterval = 10 * time.Second
	//如果没有心跳, 10 秒钟后节点就会被删除
	_NodeExpires = 10
)

var (
	//...
	etcdClient *etcd.Client
	//是否关停
	etcdClientClosed bool

	//当前进程注册的服务, 记录是为了销毁
	registers = map[string]*Service{}
	//service_name => [addr]service
	subscribes = map[string]*ResolverNode{}
)

//Service 一个独立的服务, 被注册, 被发现, 被调用
type Service struct {
	Name string `json:"name"` //服务名称
	Host string `json:"host"` //所属机器的 hostname, 不填的话, 程序会尝试自动填充
	Addr string `json:"addr"` //{ip}:{port}, 不填写 ip 则会自动填充, 或使用 hostname 替代 ip

	env string //所属环境
}

//一个节点, 用于负载均衡
type ResolverNode struct {
	dependType DependType
	lock       sync.Mutex

	dir          string
	services     []*Service
	resolverConn resolver.ClientConn
}

//Init 初始化
//Endpoints 为 Etcd 的服务地址, eg. http://xxxx:2379
func Init(Endpoints ...string) {
	var err error
	var etcdConfig = etcd.Config{
		Endpoints:   Endpoints,
		DialTimeout: 5 * time.Second,
	}
	if etcdClient, err = etcd.New(etcdConfig); err != nil {
		panic(err)
	}
}

//Register 要注册的服务
func Register(env string, s *Service) {
	if env == "" {
		panic("Register Error=Need Service Env")
	}
	if s.Name == "" {
		panic("Register Error=Please Set Service Name")
	}
	if s.Host == "" {
		s.Host, _ = os.Hostname()
	}
	addr := strings.SplitN(s.Addr, ":", 2)
	if len(addr) != 2 {
		panic("Register Error=Addr Format Invalid: {ip}:{port}")
	}
	if match, err := regexp.MatchString(`^\d+$`, addr[1]); err != nil || !match {
		panic("Register Error=Addr Format Invalid: {ip}:{port}")
	}
	if addr[0] == "localhost" || addr[0] == "127.0.0.1" || addr[0] == "0.0.0.0" {
		addr[0] = ""
	}
	if addr[0] == "" {
		addr[0] = util.GetIntranetIp()
	}
	if addr[0] == "" {
		addr[0] = s.Host
	}
	if addr[0] == "" {
		panic("Register: Error=Addr Not Exportable, " + s.Addr)
	}
	//创建服务节点
	s.env = env
	s.Addr = strings.Join(addr, ":")
	if leaseID, err := s.register(); err != nil {
		panic(fmt.Sprintf("Register: Lease Error=%s", err.Error()))
	} else {
		registers[s.Name] = s
		go s.keepalive(leaseID)
	}
}

//Close 关闭和清理
func Close() {
	tlog.Info("Close")
	etcdClientClosed = true
	for _, s := range registers {
		etcdClient.Delete(
			context.Background(),
			fmt.Sprintf(_DirectoryFormat, s.env, s.Name)+s.Addr,
		)
	}
	if err := etcdClient.Close(); err != nil {
		tlog.Error(err)
	}
}

//WaitForClose 堵塞等待关闭信号后完成关闭
func WaitForClose() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	tlog.Info("WaitForClose")
	<-sigs
	Close()
}

//挂载一个可用节点
func (this *ResolverNode) hang(s *Service) (err error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	for _, serv := range this.services {
		if serv.Addr == s.Addr {
			//如果连接中断, grpc 在重试时, 即便服务重启已经可用,
			//但连接仍可能是 TransientFailure 状态, 这是恢复连接前的状态.
			//这种状态仍然属于暂不可用的状态, 所以会因为重连而
			//收到 TransientFailure 的错误.
			return
		}
	}

	this.services = append(this.services, s)
	tlog.Infof("Hang: Node=%s%s", this.dir, s.Addr)

	this.updateResolverState()
	return
}

func (this *ResolverNode) remove(key string) {
	this.lock.Lock()
	defer func() {
		this.lock.Unlock()
		str, _ := json.Marshal(this.services)
		tlog.Debugf("grpc remove key [%s] remain [%s]", key, string(str))
	}()

	if len(this.services) == 0 {
		return
	}

	j := 0
	for _, serv := range this.services {
		if this.dir+serv.Addr == key {
			tlog.Infof("Remove: Node=%s", key)
			continue
		}
		this.services[j] = serv
		j++
	}
	this.services = this.services[0:j]

	this.updateResolverState()
	return
}

func (this *ResolverNode) updateResolverState() {
	addrs := make([]resolver.Address, len(this.services))
	for i, n := range this.services {
		addrs[i] = resolver.Address{
			Addr:       n.Addr,
			ServerName: n.Name,
		}
	}
	this.resolverConn.UpdateState(resolver.State{Addresses: addrs})
}

func (this *ResolverNode) subscribe(immediately bool) {
	resp, err := etcdClient.Get(context.TODO(), this.dir, etcd.WithPrefix())
	if err != nil {
		if immediately {
			panic(fmt.Sprintf("Subscribe: Error='%s' Dir=%s", err.Error(), this.dir))
		} else {
			tlog.Errorf("Subscribe: Error='%s' Dir=%s", err.Error(), this.dir)
		}
		return
	}
	for _, n := range resp.Kvs {
		s := new(Service)
		if err = json.Unmarshal(n.Value, s); err == nil {
			err = this.hang(s)
		}
		if err != nil {
			tlog.Infof("Subscribe: Node=%s, Error=%s", string(n.Value), err.Error())
		}
	}

	if len(this.services) == 0 {
		if immediately {
			panic(fmt.Sprintf("Subscribe: Error='%s' Nodes Was Empty", this.dir))
		} else {
			tlog.Errorf("Subscribe: Error='%s' Nodes Was Empty", this.dir)
		}
	}
}

func (this *ResolverNode) watching() {
	tlog.Infof("Watching %s\n", this.dir)
	tick := time.NewTicker(_DirectoryInterval)
	rch := etcdClient.Watch(context.TODO(), this.dir, etcd.WithPrefix())
	for {
		select {
		case <-tick.C:
			go this.subscribe(false)
		case wresp := <-rch:
			for _, ev := range wresp.Events {
				switch ev.Type {
				case etcd.EventTypeDelete:
					this.remove(string(ev.Kv.Key))

				case etcd.EventTypePut:
					var s = new(Service)
					if err := json.Unmarshal(ev.Kv.Value, s); err != nil {
						tlog.Infof("Watching: Node=%+v Parse Error=%s", ev, err.Error())
						continue
					}
					if err := this.hang(s); err == nil {
						tlog.Infof("Watching: Node=%s%s Addup", this.dir, s.Addr)
					} else {
						tlog.Infof("Watching: Node=%s%s Addup ERROR=%s", this.dir, s.Addr, err.Error())
					}

				}
			}
		}
	}
}

func (s *Service) register() (etcd.LeaseID, error) {
	tlog.Infof("Register Service=%+v", s)
	if etcdClient == nil {
		panic("Register Error=Please Init Ected Connection Firstly")
	}
	//获取串行化内容
	bin, err := json.Marshal(s)
	if err != nil {
		return 0, err
	}
	//设置 10 秒租约
	lease, err2 := etcdClient.Grant(context.TODO(), _NodeExpires)
	if err2 != nil {
		return 0, err2
	}
	k := fmt.Sprintf(_DirectoryFormat, s.env, s.Name) + s.Addr
	v := string(bin)
	w := etcd.WithLease(lease.ID)
	if _, err := etcdClient.Put(context.TODO(), k, v, w); err != nil {
		return 0, err
	}
	return lease.ID, nil
}

func (s *Service) keepalive(id etcd.LeaseID) {
	for {
		if etcdClientClosed {
			return
		}
		tlog.Infof("Keepalive Env=%s Service=%s Addr=%s", s.env, s.Name, s.Addr)
		keepRespChan, err := etcdClient.KeepAlive(context.TODO(), id)
		if err == nil {
			for keepResp := range keepRespChan {
				if keepResp == nil {
					break
				}
			}
			tlog.Infof("Keepalive Over By Channel Closed, Will Retry, Service=%+v", s)
		} else {
			time.Sleep(time.Second)
			tlog.Infof("Keepalive Retry, Service=%+v, Error=%v", s, err)
		}
		if etcdClientClosed {
			return
		}
		if id, err = s.register(); err != nil {
			tlog.Infof("Register Error=%s Service=%+v", err.Error(), s)
			time.Sleep(time.Second)
		}
	}
}
