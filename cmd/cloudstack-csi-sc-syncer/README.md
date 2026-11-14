# cloudstack-csi-sc-syncer

`cloudstack-csi-sc-syncer` connects to CloudStack (using the same CloudStack
configuration file as `cloudstack-csi-driver`), lists all disk offerings
suitable for usage in Kubernetes (currently: checks they have a custom size),
and creates corresponding Storage Classes in Kubernetes if needed.

It also adds a label to the Storage Classes it creates.

If option `-delete=true` is passed, it may also delete Kubernetes Storage
Classes, when they have its label and their corresponding CloudStack disk
offering has been deleted.

## Usage

You may use it locally or as a Kubernetes Job.

### Locally

You must have a CloudStack configuration file and a Kubernetes `kubeconfig`
file.

1. Download `cloudstack-csi-sc-syncer` from [latest release](https://github.com/leaseweb/cloudstack-csi-driver/releases/latest/);

1. Set the execution permission:

   ```
   chmod +x ./cloudstack-csi-sc-syncer
   ```

1. Then simply execute the tool:

   ```
   ./cloudstack-csi-sc-syncer <options>
   ```

Run `./cloudstack-csi-sc-syncer -h` to get the complete list of options and their default values.

### As a Kubernetes Job

You may run `cloudstack-csi-sc-syncer` as a Kubernetes Job. In that case, it
re-uses the CloudStack configuration file in Secret `cloudstack-secret`, and use
in-cluster Kubernetes authentification, using a ServiceAccount.

```sh
export version=...

kubectl apply -f - <<E0F
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloudstack-csi-sc-syncer
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cloudstack-csi-sc-syncer-role
rules:
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "create", "list", "update", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cloudstack-csi-sc-syncer-binding
subjects:
  - kind: ServiceAccount
    name: cloudstack-csi-sc-syncer
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: cloudstack-csi-sc-syncer-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: batch/v1
kind: Job
metadata:
  namespace: kube-system
  name: cloudstack-csi-sc-syncer
spec:
  template:
    spec:
      serviceAccountName: cloudstack-csi-sc-syncer
      containers:
        - name: cloudstack-csi-sc-syncer
          image: ghcr.io/leaseweb/cloudstack-csi-sc-syncer:${version}
          args:
            - "-cloudstackconfig=/etc/cloudstack-csi-driver/cloud-config"
            - "-kubeconfig=-"
             - "-nodeName=$(NODE_NAME)"
          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: cloudstack-conf
              mountPath: /etc/cloudstack-csi-driver
            - name: cloud-init-dir
              mountPath: /run/cloud-init/
      restartPolicy: Never
      volumes:
        - name: cloudstack-conf
          secret:
            secretName: cloudstack-secret
        - name: cloud-init-dir
          hostPath:
            path: /run/cloud-init/
            type: Directory
E0F
```

You may adapt the Job defined above, e.g. to create a CronJob.
