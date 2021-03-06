# ![aardvark](https://openclipart.org/image/150px/svg_to_png/23150/papapishu-Aardvark.png) aardvark

[![Mozilla Public License](https://img.shields.io/badge/license-MPL-blue.svg)](https://www.mozilla.org/MPL)
[![Go Report Card](https://goreportcard.com/badge/github.com/myENA/aardvark)](https://goreportcard.com/report/github.com/myENA/aardvark)
[![Build Status](https://travis-ci.org/myENA/aardvark.svg?branch=master)](https://travis-ci.org/myENA/aardvark)
[![Docker Pulls](https://img.shields.io/docker/pulls/myena/aardvark.svg)](https://hub.docker.com/r/myena/aardvark)
[![Docker Automated Build](https://img.shields.io/docker/automated/myena/aardvark.svg)](https://hub.docker.com/r/myena/aardvark)

## Summary

Aardvark was built to expose containers within weave networks via iBGP to upstream route reflectors
without pulling in more complex networking plugins with features we didn't need.
In addition to pushing an iBGP route advertisement aardvark will also optionally replace the containers default
route to egress out of the `weave` bridge as opposed to the `docker_gwbridge`.

The application is meant to run on every Docker host and have access to the local Docker
socket.  See the included [docker-compose.yml](docker-compose.yml) and [aardvark.nomad](aardvark.nomad) for
examples.  The expanded capabilities and `/proc` mount are only required if you want to use the defautl route
replacement functionality.

While this was built against a weave deployment, there shouldn't be anything inherently weave specific about the
application besides the example script in the usage section below.

## Building/Installing

```
git clone https://github.com/myENA/aardvark.git
cd aardvark
make docker
```

To use the latest container from Docker Hub ...

```
docker pull myena/aardvark
docker run myena/aardvark
```

To run on Nomad ...

```
git clone https://github.com/myENA/aardvark.git
cd aardvark
```

Edit the job specification file `aardvark.nomad` to suit your environment.

## Usage

### Summary

```
ahurt$ ./aardvark --help
Usage of aardvark:
  -asn uint
        local and remote peer ASN (default 65123)
  -defaultRoute string
        container default route or go-sockaddr template
  -id string
        local router ID and next-hop or go-sockaddr template (default "{{ GetPrivateIP }}")
  -network string
        watched Docker network(s) in CSV format (default "weave")
  -peer string
        upstream BGP peer(s) in CSV format
  -text
        enable plain-text logging - json if not specified

```

### Weave without MASQUERADE

> When [#3388](https://github.com/weaveworks/weave/pull/3388) is in a release version
the below script and process is no longer necessary.

In our environment we want the containers weave address to be seen by other services on the network.
In other words, we do not want the container to NAT through the host.  The current `weave expose` functionality
automatically adds `MASQUERADE` rules to the system.  We work-around this with the following script at
`/usr/local/sbin/weave-export.sh`

```bash
#!/usr/bin/env bash

## settings
MGMT_IF="eth0"
DOCKER_NET="app"

## ensure proper path
PATH=/usr/local/bin:/usr/local/sbin:$PATH

## wait for weave to be available
while [ $(weave status > /dev/null 2>&1; echo $?) -ne 0 ]; do
  sleep 0.5
done

## get last octet of first management interface address
LAST_OCTET=$(ip addr show dev ${MGMT_IF} | awk -F ' *|/' '/inet /{split($3,a,".");print a[4]}' | head -1)

## get weave network subnet from docker
WEAVE_NET=$(docker network inspect ${DOCKER_NET} -f '{{with $conf := index .IPAM.Config 0}}{{$conf.Subnet}}{{end}}')

## expose network
weave expose $(awk -v last=${LAST_OCTET} -F '/' '{split($1,a,".");print a[1] "." a[2] "." a[3] "." last "/" $2}' <<< ${WEAVE_NET})

## cleanup rules
for rule in $(iptables -t nat -L WEAVE --line-numbers | awk '/MASQUERADE |RETURN /{print $1}' | sort -rn); do
  iptables -t nat -D WEAVE ${rule}
done
```

This is run on startup via a systemd service in `/etc/systemd/system/weave-export.service`

```text
[Unit]
After=docker.service

[Service]
ExecStart=/usr/local/sbin/weave-export.sh

[Install]
WantedBy=default.target
```

This takes care of exposing the docker network (`DOCKER_NET`) via weave
using the last octec of the system's management interface (`MGMT_IFACE`) to complete the exposed address.
This in combination with aardvark running with `-defaultRoute "{{ GetInterfaceIP \"weave\" }}"` allows our
containerized applications running in a weave network to be first-class network citizens.
 