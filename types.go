package main

import (
	"github.com/nimbix/k8s-rdma-device-plugin/ibverbs"
)

type Device struct {
	RdmaDevice ibverbs.IbvDevice
	NetDevice  string
}

type SimpleRDMADevice struct {
	Name    string
	DevName string
	DevPath string
}
