job "aardvark" {
  datacenters = [
    "global"
  ]
  type = "system"
  update {
    max_parallel = 1
    auto_revert = true
  }
  group "aardvark" {
    restart {
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }
    task "aardvark" {
      driver = "docker"
      config {
        image = "myena/aardvark:latest"
        network_mode = "host"
        args = [
          "-asn", 65123,
          "-network", "weave, my-awesome-network",
          "-peer", "bgp01.domain.tld, bgp02.domain.tld",
          "-defaultRoute", "{{ GetInterfaceIP \"weave\" }}"
        ]
        cap_add = [
          "NET_ADMIN",
          "SYS_ADMIN"
        ]
        volumes = [
          "/var/run/docker.sock:/tmp/docker.sock",
          "/proc:/tmp/proc"
        ]
      }
    }
  }
}
