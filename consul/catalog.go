package consul

import (
	"reflect"

	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

func (r *Client) SyncCatalog(nodes map[string]*consulapi.CatalogNode, rancherNodes map[string]*consulapi.CatalogNode) {

	if reflect.DeepEqual(nodes, rancherNodes) {
		logrus.Info("Everything is in sync")
		return
	}

	// Compare nodes in Consul with the ones in Rancher
	for k, n := range nodes {
		if _, ok := rancherNodes[k]; !ok {
			// Node doesn't exists in Rancher, deregistering it
			_, err := r.deregisterCatalogNode(n.Node)
			if err != nil {
				logrus.Errorf("Error while deregistering node '%s': %v", n.Node.Node, err)
			}
		} else if !reflect.DeepEqual(n, rancherNodes[k]) {
			// Node exists in Rancher, update services if necessary
			r.syncCatalogNode(n, rancherNodes[k])
		}
	}

	// Compare nodes in Rancher with the ones in Consul
	for k, n := range rancherNodes {
		// Node doesn't exists in Consul, registering it
		if _, ok := nodes[k]; !ok {
			r.registerCatalogNode(n)
		}
	}
}

func (r *Client) syncCatalogNode(node *consulapi.CatalogNode, rancherNode *consulapi.CatalogNode) {

	if reflect.DeepEqual(node, rancherNode) {
		return
	}

	logrus.Infof("Syncing node %s", node.Node.Node)

	if !reflect.DeepEqual(node.Node, rancherNode.Node) {
		r.Client.Catalog().Register(
			&consulapi.CatalogRegistration{
				ID:              rancherNode.Node.ID,
				Node:            rancherNode.Node.Node,
				Address:         rancherNode.Node.Address,
				TaggedAddresses: rancherNode.Node.TaggedAddresses,
			},
			&consulapi.WriteOptions{},
		)
	}

	// Check services registered in Consul
	for k, s := range node.Services {
		if reflect.DeepEqual(s, rancherNode.Services[k]) {
			delete(rancherNode.Services, k)
			continue
		}

		//- deregister service
		if rancherNode.Services[k] == nil {
			_, err := r.deregisterCatalogService(node.Node, s)
			if err != nil {
				logrus.Errorf("Error while deregistering %s: %v", s.ID, err)
			}
		} else {
			_, err := r.registerCatalogService(node.Node, rancherNode.Services[k])
			if err != nil {
				logrus.Errorf("Error while registering %s: %v", s.ID, err)
			}
			delete(rancherNode.Services, k)
		}
	}

	// Check public services registered in Rancher
	for k, s := range rancherNode.Services {
		if reflect.DeepEqual(s, node.Services[k]) {
			continue
		}

		_, err := r.registerCatalogService(node.Node, s)
		if err != nil {
			logrus.Errorf("Error while registering %s: %v", s.ID, err)
		}
	}
}

func (r *Client) registerCatalogNode(node *consulapi.CatalogNode) {

	logrus.Infof("Registering node %s", node.Node.Node)

	for _, s := range node.Services {
		_, err := r.registerCatalogService(node.Node, s)
		if err != nil {
			logrus.Errorf("Error while registering %s: %v", s.ID, err)
		}
	}
}

func (r *Client) deregisterCatalogNode(node *consulapi.Node) (wm *consulapi.WriteMeta, err error) {

	logrus.Infof("Deregistering node %s", node.Node)

	return r.Client.Catalog().Deregister(
		&consulapi.CatalogDeregistration{
			Node: node.Node,
		},
		&consulapi.WriteOptions{},
	)
}

func (r *Client) registerCatalogService(node *consulapi.Node, service *consulapi.AgentService) (wm *consulapi.WriteMeta, err error) {

	logrus.Infof("Registering service %s on %s", service.ID, node.Node)

	return r.Client.Catalog().Register(
		&consulapi.CatalogRegistration{
			ID:              node.ID,
			Node:            node.Node,
			Address:         node.Address,
			TaggedAddresses: node.TaggedAddresses,
			Service: &consulapi.AgentService{
				ID:                service.ID,
				Service:           service.Service,
				Tags:              service.Tags,
				Port:              service.Port,
				Address:           service.Address,
				EnableTagOverride: service.EnableTagOverride,
			},
		},
		&consulapi.WriteOptions{},
	)
}

func (r *Client) deregisterCatalogService(node *consulapi.Node, service *consulapi.AgentService) (wm *consulapi.WriteMeta, err error) {

	logrus.Infof("Deregistering service %s on %s", service.ID, node.Node)

	return r.Client.Catalog().Deregister(
		&consulapi.CatalogDeregistration{
			Node:      node.Node,
			ServiceID: service.ID,
		},
		&consulapi.WriteOptions{},
	)
}
