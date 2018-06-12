# RDMA device plugin for Kubernetes 

Forked from the original repo: https://github.com/hustcat/k8s-rdma-device-plugin

## Introduction

`k8s-rdma-device-plugin` is a [device plugin](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md) for Kubernetes to manage [RDMA](https://en.wikipedia.org/wiki/Remote_direct_memory_access) devices.

This fork updates the [original k8s-rdma-device-plugin repo](https://github.com/hustcat/k8s-rdma-device-plugin) for [Kubernetes](https://kubernetes.io/) 1.10 (with limited testing on 1.11 beta). 
From version 1.9 to version 1.10, the device plugin API changed from [v1alpha](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/apis/deviceplugin/v1alpha) to [v1beta1](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/apis/deviceplugin/v1beta1).
Support is also added to use POWER8 (ppc64le) architecture along with amd64.

## Kubernetes deployment

Deploying the device plugin means deploying a DaemonSet for the cluster. The images will be pulled to match the arch of node.

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
      - image: nimbix/k8s-rdma-device-plugin:1.10
        name: rdma-device-plugin-ctr
        args: ["-log-level", "debug"]
        securityContext:
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
          add: ["ALL"]
      resources:
        limits:
          tencent.com/rdma: 1 # requesting 1 RDMA device
```

### TODO

* Update vendor