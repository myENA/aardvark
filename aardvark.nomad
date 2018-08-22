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
        network_mode = "host"
        image = "ena/aardvark:latest"
        args = [
          "-asn", 65000,
          "-network", "weave, my-awesome-network",
          "-peer", "rt-reflector-01.domain.tld, rt-reflector-02.domain.tld"
        ]
        volumes = [
          "/var/run/docker.sock:/tmp/docker.sock"
        ]
      }
      resources {
        cpu    = 100
        memory = 16
      }
    }
  }
}
