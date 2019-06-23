package consul

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// ConsulService 根据自己的需求进行的服务定制
type ConsulService struct {
	IP   string
	Port int
	Tag  []string
	Name string
}

//RegisterService 向consul中注册服务
func RegisterService(consulAddress string, service *ConsulService) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulAddress
	client, err := api.NewClient(consulConfig)
	if err != nil {
		log.Errorf("New consul client err \n: %v", err)
		return
	}

	agent := client.Agent()
	interval := time.Duration(10) * time.Second
	deregister := time.Duration(1) * time.Minute

	reg := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%v-%v-%v", service.Name, service.IP, service.Port), // 服务节点的名称
		Name:    service.Name,                                                    // 服务名称
		Tags:    service.Tag,                                                     // tag，可以为空
		Port:    service.Port,                                                    // 服务端口
		Address: service.IP,                                                      // 服务 IP
		// In Consul 0.7 and later, checks that are associated with a service
		// may also contain this optional DeregisterCriticalServiceAfter field,
		// which is a timeout in the same Go time format as Interval and TTL. If
		// a check is in the critical state for more than this configured value,
		// then its associated service (and all of its associated checks) will
		// automatically be deregistered.
		Check: &api.AgentServiceCheck{ // 健康检查
			Interval:                       interval.String(),                                               // 健康检查间隔
			GRPC:                           fmt.Sprintf("%v:%v/%v", service.IP, service.Port, service.Name), // grpc 支持，执行健康检查的地址，service 会传到 Health.Check 函数中
			DeregisterCriticalServiceAfter: deregister.String(),                                             // 注销时间，相当于过期时间
		},
	}

	log.Printf("registing to %v\n", consulAddress)
	if err := agent.ServiceRegister(reg); err != nil {
		log.Printf("Service Register error\n%v", err)
		return
	}

}
