FROM scratch
COPY bin/rancher-consul-registrator /rancher-consul-registrator
ENTRYPOINT ["/rancher-consul-registrator"]
