# Thermald-Go

![GitHub release (latest by date)](https://img.shields.io/github/v/release/machsix/thermald-go)
![GitHub](https://img.shields.io/github/license/machsix/thermald-go)
![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/machsix/thermald-go/release.yml)

Thermald-Go (aka: thermal daemon) is a thermal monitoring daemon that monitors the temperature of CPU, HDD, and NVMe.

## Installation

To install thermald-go, you can use the provided Debian package:

```sh
sudo dpkg -i thermald-go_<version>_<architecture>.deb
```

## Usage

To run thermald-go, you need to have root permission as the program calls `smartctl` and `nvme` to get HDD and NVMe temperatures.

```sh
Usage: thermald-go [--daemon] [--port PORT] [--cache CACHE] [--endpoint ENDPOINT] [--version] [--compatible]

Options:
  --daemon, -d           Run as a daemon [default: false]
  --port PORT, -p PORT   Port number [default: 7634]
  --cache CACHE, -t CACHE
                         Cache duration in seconds [default: 60]
  --endpoint ENDPOINT, -e ENDPOINT
                         Endpoint for the HTTP server [default: /]
  --version, -V          Print version and exit
  --compatible, -c       Compatible mode: using smartctl to check temperature [default: false]
  --help, -h             display this help and exit
```

In non-daemon mode, the output will be displayed in STDOUT as shown below

```
root@nas:/root/ # thermald-go 2>/dev/null
[
  {
    "type": "cpu",
    "id": "thermal_zone0",
    "model": "Intel(R) N100",
    "temperature": 27.8,
    "zone": "/sys/class/thermal/thermal_zone0/temp"
  },
  {
    "type": "hdd",
    "id": "Y5J3VB5C",
    "model": "WDC WD140EDFZ-11A0VA0",
    "temperature": 44,
    "zone": "/dev/sda"
  }
]
```

In daemon mode, the HTTP response from http://localhost:7634 will return the thermal data in JSON format.
```JSON
[
  {
    "type": "cpu",
    "id": "thermal_zone0",
    "model": "Intel(R) N100",
    "temperature": 27.8,
    "zone": "/sys/class/thermal/thermal_zone0/temp"
  },
  {
    "type": "hdd",
    "id": "Y5J3VB5C",
    "model": "WDC WD140EDFZ-11A0VA0",
    "temperature": 38,
    "zone": "/dev/sda"
  },
]
```

If you use the [Homepage project](https://github.com/gethomepage/homepage), you can add the following service widget to `services.yaml`

```yaml
Thermald-Go:
  icon: mdi-thermometer
  href: http://nas.lan:7634
  widget:
    type: customapi
    refreshInterval: 300
    # If your docker uses host's network
    # url: http://127.0.0.1:7634

    # If you created a custom bridged network at 172.17.0.1/24
    url: http://172.17.0.1:7634
    display: block
    mappings:
      - field:
          1: temperature
        format: float
        label: CPU
        suffix: "\u2103"
      - field:
          2: temperature
        suffix: "\u2103"
        format: number
        additionalField:
          field:
            2: model
        label: sda
```


## Dependencies

Thermald-Go has zero dependencies as it uses [smart.go](https://github.com/anatol/smart.go) to read S.M.A.R.T. info. If you want to run the compatability mode, you need `smartmontools`.

## License

Thermald-Go is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for more details.

## Maintainer

Maintained by Machsix (<machsix@github.com>).

## Homepage

For more information, visit the [Thermald-Go GitHub page](https://github.com/machsix/thermald-go).
