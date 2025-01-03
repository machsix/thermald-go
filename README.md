# Thermald-Go

Thermald-Go  (aka: thermal daemon) is a thermal monitoring daemon that monitors the temperature of CPU, HDD, and NVMe.

## Installation

To install thermald-go, you can use the provided Debian package. Ensure you have the necessary dependencies:

```sh
sudo apt-get install nvme-cli smartmontools
```

Then, install the thermald-go package:

```sh
sudo dpkg -i thermald-go_<version>_<architecture>.deb
```

## Usage

To run thermald-go, you need to have root permission as the program calls `smartctl` and `nvme` to get HDD and NVMe temperatures.

```sh
Usage: thermald-go [--daemon] [--port PORT] [--cache CACHE] [--endpoint ENDPOINT] [--version]

Options:
  --daemon, -d           Run as a daemon [default: false]
  --port PORT, -p PORT   Port number [default: 7634]
  --cache CACHE, -t CACHE
                         Cache duration in seconds [default: 60]
  --endpoint ENDPOINT, -e ENDPOINT
                         Endpoint for the HTTP server [default: /]
  --version, -v          Print version and exit
  --help, -h             display this help and exit
```

In non-daedaemon mode, you will get response like the following in STDOUT

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

In daemon mode, the HTTP reponse from http://localhost:7634 will be the thermal data.


## Dependencies

Thermald-Go depends on the following packages on Debian:
- `nvme-cli`
- `smartmontools`

## License

Thermald-Go is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for more details.

## Maintainer

Maintained by Machsix (<machsix@github.com>).

## Homepage

For more information, visit the [Thermald-Go GitHub page](https://github.com/machsix/thermald-go).