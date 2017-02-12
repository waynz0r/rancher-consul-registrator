package consul

import (
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/waynz0r/rancher-consul-registrator/metadata"
)

// Client can be used to query Consul API
type Client struct {
	Client *consulapi.Client
}

func NewClient(URL string, token string) *Client {

	uri, err := url.Parse(URL)
	if err != nil {
		logrus.Fatalf("Bad consul url: %s", URL)
	}

	config := consulapi.DefaultConfig()
	config.Token = token
	if uri.Scheme == "consul-unix" {
		config.Address = strings.TrimPrefix(uri.String(), "consul-")
	} else if uri.Scheme == "consul-tls" {
		tlsConfigDesc := &consulapi.TLSConfig{
			Address:            uri.Host,
			CAFile:             os.Getenv("CONSUL_CACERT"),
			CertFile:           os.Getenv("CONSUL_TLSCERT"),
			KeyFile:            os.Getenv("CONSUL_TLSKEY"),
			InsecureSkipVerify: true,
		}
		tlsConfig, err := consulapi.SetupTLSConfig(tlsConfigDesc)
		if err != nil {
			logrus.Fatalf("Cannot set up Consul TLSConfig: %s", err)
		}
		config.Scheme = "https"
		transport := cleanhttp.DefaultPooledTransport()
		transport.TLSClientConfig = tlsConfig
		config.HttpClient.Transport = transport
		config.Address = uri.Host
	} else if uri.Host != "" {
		config.Address = uri.Host
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		logrus.Fatalf("consul: %s", uri.Scheme)
	}

	return &Client{Client: client}
}

func (r *Client) Ping() (string, error) {

	status := r.Client.Status()
	leader, err := status.Leader()
	if err != nil {
		return "", err
	}

	return leader, nil
}

func (r *Client) Node(node string, q *consulapi.QueryOptions) (n *consulapi.CatalogNode, err error) {

	n, _, err = r.Client.Catalog().Node(node, q)
	if err != nil {
		return n, err
	}

	return n, nil
}

func (r *Client) Nodes(environmentUUID string, q *consulapi.QueryOptions) (nodes map[string]*consulapi.CatalogNode, err error) {

	ns, _, err := r.Client.Catalog().Nodes(q)
	if err != nil {
		return nodes, err
	}

	nodes = make(map[string]*consulapi.CatalogNode)

	for _, node := range ns {
		// Only get the nodes registered for the selected Rancher environment
		if isRancherNode(node, environmentUUID) {
			n, err := r.Node(node.Node, &consulapi.QueryOptions{})
			if err != nil {
				logrus.Errorf("%v", err)
			}
			nodes[n.Node.Address] = removeNotRancherRegisteredServices(n, environmentUUID)
		}
	}

	return nodes, nil
}

func isRancherNode(node *consulapi.Node, EnvironmentUUID string) bool {

	if _, ok := node.TaggedAddresses[sanitizeLabel("rancher-"+EnvironmentUUID+"-ip")]; ok {
		return true
	}

	return false
}

func isRancherRegisteredService(service *consulapi.AgentService, EnvironmentUUID string) bool {

	for _, tag := range service.Tags {
		if tag == sanitizeLabel("rancher-"+EnvironmentUUID) {
			return true
		}
	}

	return false
}

func removeNotRancherRegisteredServices(node *consulapi.CatalogNode, environmentUUID string) *consulapi.CatalogNode {

	for k, s := range node.Services {
		if !isRancherRegisteredService(s, environmentUUID) {
			delete(node.Services, k)
		}
	}

	return node
}

func ConvertRancherServices(services []metadata.Service) (nodes map[string]*consulapi.CatalogNode) {

	nodes = make(map[string]*consulapi.CatalogNode)

	for _, s := range services {
		if _, ok := nodes[s.IP]; !ok {
			cr := &consulapi.CatalogNode{
				Node: &consulapi.Node{
					Node:    s.HostName,
					Address: s.IP,
					TaggedAddresses: map[string]string{
						sanitizeLabel("rancher-" + s.EnvironmentUUID + "-ip"): s.IP,
						"wan": s.IP,
					},
				},
				Services: make(map[string]*consulapi.AgentService, 0),
			}
			nodes[s.IP] = cr
		}

		serviceName := s.StackName + "-" + s.Name
		serviceID := serviceName + "-" + strconv.Itoa(s.Port)

		nodes[s.IP].Services[serviceID] = &consulapi.AgentService{
			ID:                serviceID,
			Service:           serviceName,
			Port:              s.Port,
			Address:           s.IP,
			EnableTagOverride: false,
			Tags: []string{
				"created-by-rancher",
				sanitizeLabel("rancher-" + s.EnvironmentUUID),
				sanitizeLabel(s.EnvironmentName),
			},
		}
	}

	return nodes
}

func sanitizeLabel(label string) string {

	re := regexp.MustCompile("[^a-zA-Z0-9-]")
	dashes := regexp.MustCompile("[-]+")
	label = re.ReplaceAllString(label, "-")
	label = dashes.ReplaceAllString(label, "-")
	return strings.ToLower(strings.Trim(label, "-"))
}
