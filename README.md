# Leaseweb Cloudstack CSI Helm Charts
 [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Releases downloads](https://img.shields.io/github/downloads/Leaseweb/cloudstack-csi-driver/total.svg)](https://github.com/Leaseweb/cloudstack-csi-driver/releases) ![Release Charts](https://github.com/Leaseweb/cloudstack-csi-driver/actions/workflows/charts-release.yaml/badge.svg?branch=master)

This functionality is in beta and is subject to change. The code is provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.

## Usage

[Helm](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

Once Helm is set up properly, add the repository as follows:

```console
helm repo add cloudstack-csi https://leaseweb.github.io/cloudstack-csi-driver
```
### Add repo

  ```console
    helm repo add csi https://leaseweb.github.io/cloudstack-csi-driver
    helm repo update
  ```

### Install CloudStack CSI chart

  ```console
    helm install cloudstack-csi cloudstack-csi/cloudstack-csi
  ```

## License
<!-- Keep full URL links to repo files because this README syncs from main to gh-pages.  -->
[Apache 2.0 License](https://github.com/Leaseweb/cloudstack-csi-driver/blob/master/LICENSE).

## Helm charts build status

![Release Charts](https://github.com/Leaseweb/cloudstack-csi-driver/actions/workflows/charts-release.yaml/badge.svg?branch=master)