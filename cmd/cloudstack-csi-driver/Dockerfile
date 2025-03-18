FROM alpine:3.21

LABEL \
    org.opencontainers.image.description="CloudStack CSI driver" \
    org.opencontainers.image.source="https://github.com/leaseweb/cloudstack-csi-driver/"

RUN apk add --no-cache \
    ca-certificates \
    # Provides mkfs.ext2, mkfs.ext3, mkfs.ext4 (used by k8s.io/mount-utils)
    e2fsprogs \
    e2fsprogs-extra \
    # Provides mkfs.xfs
    xfsprogs \
    # blkid, mount and umount are required by k8s.io/mount-utils \
    blkid \
    mount \
    umount

COPY ./bin/cloudstack-csi-driver /cloudstack-csi-driver
ENTRYPOINT ["/cloudstack-csi-driver"]