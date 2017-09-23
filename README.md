# service-center 
[![Build Status](https://travis-ci.org/ServiceComb/service-center.svg?branch=master)](https://travis-ci.org/ServiceComb/service-center)   [![Coverage Status](https://coveralls.io/repos/github/ServiceComb/service-center/badge.svg?branch=master)](https://coveralls.io/github/ServiceComb/service-center?branch=master)  [![Go Report Card](https://goreportcard.com/badge/github.com/ServiceComb/service-center)](https://goreportcard.com/report/github.com/ServiceComb/service-center)  [![GoDoc](https://godoc.org/github.com/ServiceComb/service-center?status.svg)](https://godoc.org/github.com/ServiceComb/service-center)

A standalone service center allows services to register their instance information and to discover providers of a given service.

## Quick Start

### Getting Service Center

The easiest way to get Service Center is to use one of the pre-built release binaries which are available for Linux, Windows and Docker. Instructions for using these binaries are on the [GitHub releases page][github-release].

[github-release]: https://github.com/servicecomb/service-center/releases/

### Building and Running Service Center

You don't need to build from source to use Service Center (binaries on the [GitHub releases page][github-release]).When you get these binaries, you can execute the start script to run Service Center.

Windows(service-center-xxx-windows-amd64.zip):
```
start.bat
```

Linux(service-center-xxx-linux-amd64.tar.gz):
```sh
./start.sh
```
Docker:
```sh
docker pull servicecomb/service-center
docker run -d -p 30100:30100 servicecomb/service-center
```


##### If you want to try out the latest and greatest, Service Center can be easily built. 

Download the Code
```sh
git clone https://github.com/ServiceComb/service-center.git $GOPATH/src/github.com/ServiceComb/service-center
cd $GOPATH/src/github.com/ServiceComb/service-center
```

Dependencies

We use gvt for dependency management, please follow below steps to download all the dependency.
```sh
go get github.com/FiloSottile/gvt
gvt restore
```
If you face any issue in downloading the dependency because of insecure connection then you can use ```gvt restore -precaire```

Build the Service-Center

```sh
go build -o service-center
```

First, you need to run a etcd(version: 3.x) as a database service and then modify the etcd IP and port in the Service Center configuration file (./etc/conf/app.conf : manager_cluster).

```sh
wget https://github.com/coreos/etcd/releases/download/v3.1.8/etcd-v3.1.8-linux-amd64.tar.gz
tar -xvf etcd-v3.1.8-linux-amd64.tar.gz
cd etcd-v3.1.8-linux-amd64
./etcd

cd $GOPATH/src/github.com/ServiceComb/service-center
cp -r ./etc/conf .
./service-center
```
This will bring up Service Center listening on ip/port 127.0.0.1:30100 for service communication.If you want to change the listening ip/port, you can modify it in the Service Center configuration file (./conf/app.conf : httpaddr,httpport).

[github-release]: https://github.com/servicecomb/service-center/releases/

## Documentation

Project documentation is available on the [ServiceComb website][servicecomb-website]. You can also find some development guide [here](/docs)

[servicecomb-website]: http://servicecomb.io/
      
## Contact

Bugs: [issues](https://github.com/servicecomb/service-center/issues)

## Contributing

See [Contribution guide](/docs/contribution.md) for details on submitting patches and the contribution workflow.

## Reporting Issues

See reporting bugs for details about reporting any issues.
