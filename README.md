# RDMA device plugin for Kubernetes 

Forked from the original repo: https://github.com/hustcat/k8s-rdma-device-plugin

## Introduction

`k8s-rdma-device-plugin` is a [device plugin](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md) for Kubernetes to manage [RDMA](https://en.wikipedia.org/wiki/Remote_direct_memory_access) devices.

This fork updates the [original k8s-rdma-device-plugin repo](https://github.com/hustcat/k8s-rdma-device-plugin) for [Kubernetes](https://kubernetes.io/) 1.10 (with limited testing on 1.11 beta). From version 1.9 to version 1.10, the device plugin API changed from [v1alpha](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/apis/deviceplugin/v1alpha) to [v1beta1](https://github.com/kubernetes/kubernetes/tree/master/pkg/kubelet/apis/deviceplugin/v1beta1).
