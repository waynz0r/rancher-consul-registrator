package metadata

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher-metadata/metadata"
)

type Client struct {
	Client          metadata.Client
	EnvironmentName string
	EnvironmentUUID string
}

type Service struct {
	Name            string
	StackName       string
	EnvironmentName string
	EnvironmentUUID string
	HostName        string
	IP              string
	Port            int
}

func NewClient(metadataURL string) (*Client, error) {
	m, err := metadata.NewClientAndWait(metadataURL)
	if err != nil {
		logrus.Fatalf("Failed to configure rancher-metadata: %v", err)
	}

	envName, envUUID, err := getEnvironment(m)
	if err != nil {
		logrus.Fatalf("Error reading stack info: %v", err)
	}

	return &Client{
		Client:          m,
		EnvironmentName: envName,
		EnvironmentUUID: envUUID,
	}, nil
}

func getEnvironment(m metadata.Client) (string, string, error) {
	timeout := 30 * time.Second
	var err error
	var stack metadata.Stack
	for i := 1 * time.Second; i < timeout; i *= time.Duration(2) {
		stack, err = m.GetSelfStack()
		if err != nil {
			logrus.Errorf("Error reading stack info: %v...will retry", err)
			time.Sleep(i)
		} else {
			return stack.EnvironmentName, stack.EnvironmentUUID, nil
		}
	}
	return "", "", fmt.Errorf("Error reading stack info: %v", err)
}

func (m *Client) GetCerts() (certs map[string]string, err error) {

	s, err := m.Client.GetSelfService()
	if err != nil {
		return certs, err
	}

	certs = make(map[string]string)

	names := []string{"ca.crt", "client.crt", "client.key"}
	for _, name := range names {
		if s.Metadata[name] != nil {
			certs[name] = s.Metadata[name].(string)
		}
	}

	return certs, nil
}

func (m *Client) GetVersion() (string, error) {
	return m.Client.GetVersion()
}

func (m *Client) Services(self bool) (services []Service, err error) {

	containers, err := m.Client.GetContainers()
	if err != nil {
		return services, err
	}

	for _, container := range containers {
		if len(container.ServiceName) == 0 || len(container.Ports) == 0 || !containerStateOK(container) {
			continue
		}

		if self {
			selfHost, _ := m.Client.GetSelfHost()
			if selfHost.UUID != container.HostUUID {
				continue
			}
		}

		hostUUID := container.HostUUID
		if len(hostUUID) == 0 {
			logrus.Debugf("Container's %v host_uuid is empty", container.Name)
			continue
		}

		host, err := m.Client.GetHost(hostUUID)
		if err != nil {
			logrus.Errorf("%v", err)
			continue
		}

		ip, ok := host.Labels["io.rancher.host.external_dns_ip"]

		if !ok || ip == "" {
			ip = host.AgentIP
		}

		// Register the host itself as a service
		services = append(services, Service{
			Name:            "host",
			StackName:       "rancher",
			EnvironmentName: m.EnvironmentName,
			EnvironmentUUID: m.EnvironmentUUID,
			HostName:        host.Name,
			IP:              ip,
		})

		for _, portDef := range container.Ports {
			port, err := strconv.Atoi(strings.Split(portDef, ":")[1])
			if err != nil {
				logrus.Errorf("%v", err)
				continue
			}

			services = append(services, Service{
				Name:            container.ServiceName,
				StackName:       container.StackName,
				EnvironmentName: m.EnvironmentName,
				EnvironmentUUID: m.EnvironmentUUID,
				HostName:        host.Name,
				Port:            port,
				IP:              ip,
			})
		}
	}

	return services, nil
}

func containerStateOK(container metadata.Container) bool {
	switch container.State {
	case "running":
	default:
		return false
	}

	switch container.HealthState {
	case "healthy":
	case "initializing":
	case "updating-healthy":
	case "":
	default:
		return false
	}

	return true
}
