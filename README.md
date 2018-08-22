[![Mozilla Public License](https://img.shields.io/badge/license-MPL-blue.svg)](https://www.mozilla.org/MPL)
[![Go Report Card](https://goreportcard.com/badge/github.com/myENA/aardvark)](https://goreportcard.com/report/github.com/myENA/aardvark)
[![Build Status](https://travis-ci.org/myENA/aardvark.svg?branch=master)](https://travis-ci.org/myENA/aardvark)
[![Downloads](https://img.shields.io/github/downloads/myENA/aardvark/total.svg)](https://github.com/myENA/aardvark/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/myena/aardvark.svg)](https://hub.docker.com/r/myena/aardvark)
[![Docker Automated Build](https://img.shields.io/docker/automated/myena/aardvark.svg)](https://hub.docker.com/r/myena/aardvark)

# aardvark

## Summary

Aardvark was built to allow us to expose containers within weave networks via iBGP to
upstream route reflectors without pulling in more complex networking plugins with features
we didn't need.

The application is meant to run on every Docker host and have access to the local Docker
socket.  See the included [docker-compose.yml](docker-compose.yml) and [aardvark.nomad](aardvark.nomad) for
examples.

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
        local and remote peer ASN (default 65321)
  -id string
        local router ID (IP address) or go-sockaddr template (default "{{ GetPrivateIP }}")
  -network string
        watched Docker network(s) in CSV format (default "weave")
  -peer string
        upstream BGP peer(s) in CSV format
  -text
        enable plain-text logging - json if not specified
```
