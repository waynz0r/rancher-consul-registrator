package consul

import (
	"reflect"

	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

func (r *Client) AgentServices(environmentUUID string) (services map[string]*consulapi.AgentService, err error) {

	s, err := r.Client.Agent().Services()
	if err != nil {
		return s, err
	}

	services = make(map[string]*consulapi.AgentService)

	for k, service := range s {
		if isRancherRegisteredService(service, environmentUUID) {
			services[k] = service
		}
	}

	return services, nil
}

func (r *Client) SyncAgentServices(environmentUUID string, rancherNodes map[string]*consulapi.CatalogNode) {

	agentServices, err := r.AgentServices(environmentUUID)
	if err != nil {
		logrus.Errorf("Error while getting services from Consul: %v", err)
		return
	}

	for _, n := range rancherNodes {
		if reflect.DeepEqual(agentServices, n.Services) {
			logrus.Info("Everything is in sync")
			continue
		}

		// Check services registered in Consul
		for k, s := range agentServices {
			if reflect.DeepEqual(s, n.Services[k]) {
				delete(n.Services, k)
				continue
			}

			//- deregister service
			if n.Services[k] == nil {
				err := r.deregisterAgentService(s)
				if err != nil {
					logrus.Errorf("Error while deregistering %s: %v", s.ID, err)
				}
			} else {
				err := r.registerAgentService(n.Services[k])
				if err != nil {
					logrus.Errorf("Error while registering %s: %v", s.ID, err)
				}
				delete(n.Services, k)
			}
		}

		// Check public services registered in Rancher
		for k, s := range n.Services {
			if reflect.DeepEqual(s, agentServices[k]) {
				continue
			}

			err := r.registerAgentService(s)
			if err != nil {
				logrus.Errorf("Error while registering %s: %v", s.ID, err)
			}
		}
	}
}

func (r *Client) registerAgentService(service *consulapi.AgentService) (err error) {

	logrus.Infof("Registering service %s", service.ID)

	return r.Client.Agent().ServiceRegister(
		&consulapi.AgentServiceRegistration{
			ID:                service.ID,
			Name:              service.Service,
			Tags:              service.Tags,
			Port:              service.Port,
			Address:           service.Address,
			EnableTagOverride: service.EnableTagOverride,
		},
	)
}

func (r *Client) deregisterAgentService(service *consulapi.AgentService) (err error) {

	logrus.Infof("Deregistering agent service %s", service.ID)

	return r.Client.Agent().ServiceDeregister(service.ID)
}
