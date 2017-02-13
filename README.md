# Rancher Consul Registrator

Simple service for Rancher to register/deregister services to Consul.

## How to use

There are two distinct modes of operation, local and global.

Local mode assumes that a local Consul agent is running on every host in Rancher, so the service has to run on every hosts in a Rancher environment and will talk to the local Consul agent running on the hosts.

In global mode the service talks to a remote Consul API and registers/deregisters all hosts and public services of the Rancher envinronment to that cluster.

## Getting it

Get the latest release, master, or any version of Rancher Consul Registrator via [Docker Hub](https://registry.hub.docker.com/u/waynz0r/rancher-consul-registrator/)
