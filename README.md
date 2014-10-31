# topod
[![Build Status](https://travis-ci.org/wlsailor/topod.svg?branch=master)](https://travis-ci.org/wlsailor/topod)

Topod is a simple PAAS tools, focus on light weight configuration management and service discovery :
* Keep your configuration files up-to-date by watching changes of remote storage like etcd
* View and modify central congifuration for your projects, get a global scene for your mess configuration
* When central configuration changed , topod can reload your process by excurte some shell commands
* Discover service and collection serivce description by scannning some ports range you have customed, and automatic modify target keys to change some process config file who cares about these service

## Current stable version: 0.5
* Watch or generate configration once

## TODO
* Service discovery and register
* Central configuration view and edit

## Getting Started
* [download and install topod](docs/installation.md)
* [quick start guide](docs/quick-start-guide.md)
* You can type ./topod -h to read more about command options

## Next steps

Check out the [docs directory](docs) for more docs.

