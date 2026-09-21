# CloudStack CSI Driver

[![Go Reference](https://pkg.go.dev/badge/github.com/leaseweb/cloudstack-csi-driver.svg)](https://pkg.go.dev/github.com/leaseweb/cloudstack-csi-driver)
[![Go Report Card](https://goreportcard.com/badge/github.com/leaseweb/cloudstack-csi-driver)](https://goreportcard.com/report/github.com/leaseweb/cloudstack-csi-driver)
[![Release](https://github.com/leaseweb/cloudstack-csi-driver/workflows/Release/badge.svg?branch=master)](https://github.com/leaseweb/cloudstack-csi-driver/actions)

This repository provides a [Container Storage Interface (CSI)](https://github.com/container-storage-interface/spec)
plugin for [Apache CloudStack](https://cloudstack.apache.org/).

## Usage with Kubernetes

### Requirements

- Minimal Kubernetes version: v1.31

- The Kubernetes cluster must run in CloudStack. Tested only in a KVM zone.

- A disk offering with custom size must be available, with type "shared".

- In order to match the Kubernetes node and the CloudStack instance,
  they should both have the same name. If not, it is also possible to use
  [cloud-init instance metadata](https://cloudinit.readthedocs.io/en/latest/topics/instancedata.html)
  to get the instance name: if the node has cloud-init enabled, metadata will
  be available in `/run/cloud-init/instance-data.json`; you should then make
  sure that `/run/cloud-init/` is mounted from the node.

- Kubernetes nodes must be in the Root domain, and be created by the CloudStack
  account whose credentials are used in [configuration](#configuration).

### Configuration

Create the CloudStack configuration file `cloud-config`.

It should have the following format, defined for the [CloudStack Kubernetes Provider](https://github.com/apache/cloudstack-kubernetes-provider):

```ini
[Global]
api-url = <CloudStack API URL>
api-key = <CloudStack API Key>
secret-key = <CloudStack API Secret>
ssl-no-verify = <Disable SSL certificate validation: true or false (optional)>
project-id = <CloudStack project ID (optional)>
```

Create a secret named `cloudstack-secret` in namespace `kube-system`:

```
kubectl create secret generic \
  --namespace kube-system \
  --from-file ./cloud-config \
  cloudstack-secret
```

If you have also deployed the [CloudStack Kubernetes Provider](https://github.com/apache/cloudstack-kubernetes-provider),
you may use the same secret for both tools.

### Deployment

```
kubectl apply -f https://github.com/leaseweb/cloudstack-csi-driver/releases/latest/download/manifest.yaml
```

### Creation of Storage classes

#### Manually

A storage class can be created manually: see [example](./examples/k8s/0-storageclass.yaml).

The `provisioner` value must be `csi.cloudstack.apache.org`.

The `volumeBindingMode` must be `WaitForFirstConsumer`, in order to delay the
binding and provisioning of a PersistentVolume until a Pod using the
PersistentVolumeClaim is created. It enables the provisioning of volumes
in respect to topology constraints (e.g. volume in the right zone).

The storage class must also have a parameter named
`csi.cloudstack.apache.org/disk-offering-id` whose value is the CloudStack disk
offering ID.

#### Using cloudstack-csi-sc-syncer

The tool `cloudstack-csi-sc-syncer` may also be used to synchronize CloudStack
disk offerings to Kubernetes storage classes.

[More info...](./cmd/cloudstack-csi-sc-syncer/README.md)

### Usage

Example:

```
kubectl apply -f ./examples/k8s/pvc.yaml
kubectl apply -f ./examples/k8s/pod.yaml
```

## Building

To build the driver binary:

```
make build-cloudstack-csi-driver
```

To build the container images:

```
make container
```

## See also

- [CloudStack Kubernetes Provider](https://github.com/apache/cloudstack-kubernetes-provider) - Kubernetes Cloud Controller Manager for Apache CloudStack
- [CloudStack documentation on storage](http://docs.cloudstack.apache.org/en/latest/adminguide/storage.html)
- [CSI (Container Storage Interface) specification](https://github.com/container-storage-interface/spec)

---

    Copyright 2021 Apalia SAS

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

            http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
