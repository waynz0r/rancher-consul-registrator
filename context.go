package main

import (
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/waynz0r/rancher-consul-registrator/consul"
	"github.com/waynz0r/rancher-consul-registrator/metadata"
)

type Context struct {
	Rancher *metadata.Client
	Consul  *consul.Client
}

// InitContext initializes the application context from environmental variables
func (c *Context) InitContext() {
	var err error

	// Initialize Rancher metadata client
	c.Rancher, err = metadata.NewClient(metadataURL)
	if err != nil {
		logrus.Fatalf("Failed to configure rancher-metadata client: %v", err)
	}
	logrus.Info("Rancher Metadata is reachable")

	certs, err := c.Rancher.GetCerts()
	if err != nil {
		logrus.Fatalf("Failed to get TLS certs from metadata: %v", err)
	}
	if len(certs) == 3 {
		DumpCerts(certs)
	}

	if localMode {
		logrus.Info("Running in local mode!")

		uri, err := url.Parse(consulURL)
		if err != nil {
			logrus.Fatalf("Bad consul url: %s", consulURL)
		}

		if ok, _ := regexp.MatchString("^RancherHostIP", uri.Host); ok {
			re := regexp.MustCompile("^RancherHostIP")
			rancherHost, _ := c.Rancher.Client.GetSelfHost()
			uri.Host = re.ReplaceAllString(uri.Host, rancherHost.AgentIP)
			consulURL = uri.String()
		}
	} else {
		logrus.Info("Running in remote mode!")
	}

	// Initialize Consul client
	c.Consul = consul.NewClient(consulURL, consulToken)
	consulLeader, err := c.Consul.Ping()
	if err != nil {
		logrus.Fatalf("Failed to configure Consul API client: %v", err)
	}
	logrus.Infof("Consul API is reachable (leader is at %s)", consulLeader)

	logrus.Infof("Sync interval set to %v seconds", syncInterval.Seconds())
}

func (c *Context) Sync(local bool) {

	logrus.Debug("Syncing public services in Rancher...")

	// Get public services from rancher
	services, err := c.Rancher.Services(local)
	if err != nil {
		logrus.Fatalf("Failed to get services: %v", err)
	}

	if local {
		// Sync
		c.Consul.SyncAgentServices(c.Rancher.EnvironmentUUID, consul.ConvertRancherServices(services))
	} else {
		// Get Consul nodes registered for this Rancher environment
		nodes, err := c.Consul.Nodes(c.Rancher.EnvironmentUUID, &consulapi.QueryOptions{})
		if err != nil {
			logrus.Fatalf("Failed to get nodes: %v", err)
		}

		// Sync
		c.Consul.SyncCatalog(nodes, consul.ConvertRancherServices(services))
	}
}

func (c *Context) Run() {

	go c.startHealthcheck()

	var wg sync.WaitGroup
	done := make(chan struct{})

	go func() {
		wg.Add(1)
		for {
			c.Sync(localMode)
			select {
			case <-time.After(syncInterval):
			case <-done:
				wg.Done()
				return
			}
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	logrus.Infof("Shutdown signal received, exiting...")
	close(done)
	wg.Wait()
}
