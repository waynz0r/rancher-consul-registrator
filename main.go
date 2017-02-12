package main

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/namsral/flag"
)

var (
	metadataURL    string
	consulURL      string
	consulToken    string
	certDir        string
	syncInterval   time.Duration
	healtcheckPort int
	localMode      bool
)

func init() {
	flag.StringVar(&metadataURL, "metadata-url", "http://rancher-metadata.rancher.internal/latest", "Rancher metadata URL")
	flag.StringVar(&consulURL, "consul-url", "consul://RancherHostIP:8500", "Consul API URL")
	flag.StringVar(&consulToken, "consul-token", "", "Consul client token")
	flag.StringVar(&certDir, "cert-dir", "/", "Where to dump the cert files from Rancher metadata")
	flag.DurationVar(&syncInterval, "sync-interval", (10 * time.Second), "Time duration between service syncs")
	flag.IntVar(&healtcheckPort, "healthcheck-port", 10000, "HTTP healthcheck port")
	flag.BoolVar(&localMode, "local-mode", true, "Only sync to local agent or register everything to remote Consul API")
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	logrus.SetOutput(os.Stdout)
}

func main() {

	flag.Parse()

	logrus.Info("Starting Consul Service Registrator")

	context := &Context{}
	context.InitContext()
	context.Run()

	os.Exit(0)
}
