# RDMA device plugin for Kubernetes 

Forked from the original repo: https://github.com/hustcat/k8s-rdma-device-plugin

## Introduction

`k8s-rdma-device-plugin` is a [device plugin](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md) for Kubernetes to manage [RDMA](https://en.wikipedia.org/wiki/Remote_direct_memory_access) devices.

This fork updates the [original k8s-rdma-device-plugin repo](https://github.com/hustcat/k8s-rdma-device-plugin) for [Kubernetes](https://kubernetes.io/) 1.11. 
From version 1.9 to version 1.10, the device plugin API changed from [v1alpha](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/apis/deviceplugin/v1alpha) to [v1beta1](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/apis/deviceplugin/v1beta1).
Support is also added to use POWER8 (ppc64le) architecture along with amd64.





## Building the plugin 

The plugin binary can be built locally with the following requirements or with the Dockerfiles that wrap the build environment and create the deployable container for Kubernetes

#### Requirements

* libibverbs
* golang 1.10+

The plugin binary can be built directly on a linux system with: `go build .` or `./build`

The container is built with: `docker build -t nimbix/k8s-rdma-device-plugin-amd64:1.11 .`

### Manually run the plugin

Run the local binary (in bin/): `k8s-rdma-device-plugin --log-level debug`

or run the container locally:

`docker run --security-opt=no-new-privileges --cap-drop=ALL --network=host -it -v /var/lib/kubelet/device-plugins:/var/lib/kubelet/device-plugins --rm nimbix/k8s-rdma-device-plugin:1.11 -log-level debug`

## Kubernetes deployment

Deploying the device plugin means deploying a Kubernetes DaemonSet for the cluster. The images will be pulled to match the architecture of node.

```
$ kubectl -n kube-system apply -f rdma-device-plugin.yml
```

### Sample YAML

A DaemonSet template to select the image and tag:

```yaml
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  name: rdma-device-plugin-daemonset
  namespace: kube-system
spec:
  template:
    metadata:
      # Mark this pod as a critical add-on; when enabled, the critical add-on scheduler
      # reserves resources for critical add-on pods so that they can be rescheduled after
      # a failure.  This annotation works in tandem with the toleration below.
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
      labels:
        name: rdma-device-plugin-ds
    spec:
      tolerations:
      # Allow this pod to be rescheduled while the node is in "critical add-ons only" mode.
      # This, along with the annotation above marks this pod as a critical add-on.
      - key: CriticalAddonsOnly
        operator: Exists
      hostNetwork: true
      containers:
      - image: nimbix/k8s-rdma-device-plugin:1.11
        name: rdma-device-plugin-ctr
        args: ["-log-level", "debug"]
        securityContext:
          # SELinux option needs to be set for RHEL OS
          seLinuxOptions:
            type: "container_runtime_t"
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
        volumeMounts:
          - name: device-plugin
            mountPath: /var/lib/kubelet/device-plugins
      volumes:
        - name: device-plugin
          hostPath:
            path: /var/lib/kubelet/device-plugins
```

This will require a Pod template with correct arch (if ppc64le is desired) for leveraging the RDMA plugin as an example:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: rdma-pod
spec:
  containers:
    - name: rdma-container
      image: nimbix/ubuntu-desktop-ppc64le:xenial
      command: ["/bin/sh"]
      args: ["-c", "sleep 999"]
      securityContext:
        capabilities:
          add: IPC_LOCK
      resources:
        limits:
          tencent.com/rdma: 1 # requesting 1 RDMA device
```

### Changelog

* Update to 1.11 Kubernetes
* Change vendoring from glide to dep
* Update the plugin container to use bionic instead of xenial, better matching on host OFED & drivers
* Merge in SELinux fix from PR

### TODO

* refactor out the ibverbs code
* drop the SRIOV and network code
* refactor out the logrus code for normal logging