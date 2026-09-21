# Changelog

All notable changes to the CloudStack CSI driver are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Dates are the dates the release tag was created.

The Helm chart is versioned and released separately under `cloudstack-csi-*`
tags; its version is bumped after the driver release it ships, so a chart
version does not line up one-to-one with a driver version.

Routine CI, linter and transitive dependency bumps are omitted. Kubernetes
library bumps are listed, since they determine which Kubernetes releases the
driver is built and tested against.

## [v0.14.0] - 2026-09-21

### Changed

- Updated Kubernetes libraries to `v0.36.4` (Kubernetes 1.36).
- Updated the CSI sidecar containers in `deploy/k8s` to csi-provisioner v6.2.1,
  csi-attacher v4.12.0, csi-resizer v2.2.1, csi-node-driver-registrar v2.17.0
  and livenessprobe v2.19.0.
- Updated the CSI spec to `v1.13.0` and `csi-test` to `v5.6.0`. These move
  together: `csi-test` v5.4.0 references capability constants that spec v1.13.0
  removes.
- Bumped the remaining direct Go dependencies (`testify`, `golang.org/x/sys`,
  `golang.org/x/text`, `k8s.io/utils`, `grpc`).
- Raised the documented minimum Kubernetes version from v1.25 to v1.31.
- External e2e test binaries now pull Kubernetes v1.36.4 instead of v1.34.2.
- Dependabot now only raises Go module PRs for direct dependencies. Note this
  also suppresses automatic security PRs for indirect modules; those alerts must
  be picked up from the repository's Dependabot alerts view.

## [v0.13.0] - 2026-08-27

### Changed

- Kubernetes 1.36 support: Kubernetes libraries updated to `v0.35.8`.
- Updated the CSI sidecar containers for Kubernetes 1.36.
- Build moved to Go 1.26 and golangci-lint 2.13.1.

### Fixed

- The Go version in `go.mod` was not bumped alongside the toolchain.

## [v0.12.1] - 2026-04-13

### Changed

- Rolled `cloudstack-go` back to v2.17.1. Newer `cloudstack-go` releases are not
  compatible with the older CloudStack version this driver runs against.

## [v0.12.0] - 2026-04-08

### Changed

- Kubernetes 1.35 support: Kubernetes libraries updated to `v0.34.3`.
- Updated the CSI spec to v1.12.0.
- Updated golangci-lint to 2.7.2.

### Fixed

- Added the missing RBAC permissions for `volumeattributesclasses`.

## [v0.11.0] - 2025-10-29

### Changed

- Kubernetes 1.33 support: Kubernetes libraries updated to `v0.33.3`.
- Build moved to Go 1.24.

## [v0.10.0] - 2025-07-30

### Added

- Support for the well-known topology labels, with improved topology handling
  overall.

### Changed

- Kubernetes 1.32 support: Kubernetes libraries updated to `v0.32.7`.
- Updated the CSI sidecar containers for Kubernetes 1.32.

## [v0.9.1] - 2025-07-22

### Fixed

- Volume expansion failed for volumes owned by a CloudStack project.

## [v0.9.0] - 2025-05-08

### Changed

- Kubernetes 1.31 support: Kubernetes libraries updated to `v0.31.8`.
- Updated the CSI spec to v1.10.0.
- Build moved to Go 1.23 and golangci-lint v1.63.4.

## [v0.8.1] - 2024-11-25

### Changed

- Kubernetes libraries updated to `v0.30.7`.
- Refactored the driver, node, mounter and controller to make them testable,
  renamed the cloud interface and added a mock cloud package with initial tests.
- Reduced the complexity of `NodePublishVolume`.

### Fixed

- The maximum attached volumes per node limit was never actually enforced.
- `NodePublishVolume` is now idempotent.
- Restored locking on node publish/unpublish, which had been removed in v0.5.0.
- Corrected the volume path reported in node volume stats.

## [v0.8.0] - 2024-10-14

### Changed

- Kubernetes 1.30 support: Kubernetes libraries updated to `v0.30.5` and the CSI
  libraries to v0.18.1.
- Bumped the CSI sidecar container versions.
- CI moved to Ubuntu 24.04 and Go 1.22.

## [v0.7.0] - 2024-09-04

### Added

- CloudStack project support.
- Issue and pull request templates, and a chart lint job in CI.

### Changed

- Kubernetes libraries updated to `v0.29.8`.

### Fixed

- `listVirtualMachine` calls now set `listall`, so nodes are found across the
  whole domain rather than only the caller's account.
- Corrected syntax errors in the syncer job chart template and fixed the syncer
  tolerations.

## [v0.6.1] - 2024-07-24

### Changed

- Complete refactor of the Helm chart, released as chart 2.0.0.
- The node DaemonSet now uses its own ServiceAccount in the deploy manifests.

### Fixed

- Added the missing tolerations to the syncer job.

## [v0.6.0] - 2024-07-01

### Added

- The driver can run as controller-only or node-only, with a flag for the
  maximum number of volumes per node.

### Changed

- Reworked the node-to-mount interface and dropped the unused version argument
  from the identity server.

### Fixed

- Corrected the topology `HostID` key and value.
- Removed the cacert volume and `dnsPolicy` from the node DaemonSet in the
  chart.
- Fixed and improved the sanity tests.

## [v0.5.0] - 2024-06-21

### Added

- Support for reading instance metadata from Ignition, alongside cloud-init.
- `priorityClassName` is set in the deploy manifests, and can be set separately
  for the controller and the node DaemonSet in the chart.
- Contextual logging throughout the driver.

### Fixed

- Reworked how operation locks are implemented and improved volume expansion.
- Corrected the "not mounted" detection so it no longer reports false positives.
- Removed locking from node publish/unpublish operations.

## [v0.4.2] - 2024-06-18

### Changed

- All chart components can now be enabled or disabled individually.
- Added a DNS policy for the node agents.

### Fixed

- Updated RBAC to match the RBAC required by the current CSI sidecar manifests.

## [v0.4.1] - 2024-03-21

### Changed

- Kubernetes libraries updated to `v0.29.3`.

## [v0.4.0] - 2024-03-13

### Added

- The Helm chart, published as chart 1.0.0.

### Changed

- Kubernetes 1.29 support: Kubernetes libraries updated to `v0.29.2`.

### Fixed

- `ssl-no-verify` is now handled correctly.

## [v0.3.0] - 2024-01-09

### Added

- Volume expansion support, with an `allowVolumeExpansion` option in the
  storage class syncer.
- A liveness probe for the node registrar.

### Changed

- Kubernetes libraries updated to `v0.29.0` and the CSI spec to v1.9.0.

## [v0.2.0] - 2023-08-28

### Added

- Leader election and a liveness probe for the controller.
- Locking around all volume operations.

### Changed

- Kubernetes libraries updated to `v0.27.5` and the CSI spec to v1.8.0.
- e2e tests now run against the Kubernetes v1.27.5 test suite.

## [v0.1.1] - 2023-06-14

### Fixed

- Reverted passing `vmID` to `DetachVolume`, which broke detach.

## [v0.1.0] - 2023-06-13

Initial release, built against Kubernetes libraries `v0.25.10` and CSI spec
v1.6.0.

[v0.14.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.13.0...v0.14.0
[v0.13.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.12.1...v0.13.0
[v0.12.1]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.12.0...v0.12.1
[v0.12.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.11.0...v0.12.0
[v0.11.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.10.0...v0.11.0
[v0.10.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.9.1...v0.10.0
[v0.9.1]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.9.0...v0.9.1
[v0.9.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.8.1...v0.9.0
[v0.8.1]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.8.0...v0.8.1
[v0.8.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.7.0...v0.8.0
[v0.7.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.6.1...v0.7.0
[v0.6.1]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.5.0...v0.6.0
[v0.5.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.4.2...v0.5.0
[v0.4.2]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.4.1...v0.4.2
[v0.4.1]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.1.1...v0.2.0
[v0.1.1]: https://github.com/leaseweb/cloudstack-csi-driver/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/leaseweb/cloudstack-csi-driver/releases/tag/v0.1.0
