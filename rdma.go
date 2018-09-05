package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"golang.org/x/net/context"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/nimbix/k8s-rdma-device-plugin/ibverbs"
)

const RdmaDeviceResource = "/sys/class/infiniband/%s/device/resource"
const NetDeviceResource = "/sys/class/net/%s/device/resource"

//func GetDevices(masterNetDevice string) ([]Device, error) {
//func GetDevices() ([]Device, error) {
//	return getAllRdmaDevices()
//
//}

// Return all the relevant Infiniband RDMA devices, excluding the SR-IOV network devices
//func getAllRdmaDevices() ([]Device, error) {
func GetDevices() ([]Device, error) {
	var devs []Device
	// Get all RDMA device list
	ibvDevList, err := ibverbs.IbvGetDeviceList()
	if err != nil {
		return nil, err
	}

	netDevList, err := GetAllNetDevice()
	if err != nil {
		return nil, err
	}
	for _, d := range ibvDevList {
		for _, n := range netDevList {
			dResource, err := getRdmaDeviceResource(d.Name)
			if err != nil {
				continue
			}
			nResource, err := getNetDeviceResource(n)
			if err != nil {
				continue
			}

			// the same device
			if bytes.Compare(dResource, nResource) == 0 {
				devs = append(devs, Device{
					RdmaDevice: d,
					NetDevice:  n,
				})
			}
		}
	}
	return devs, nil
}

// Parse the output of ibstat -- in hopes that this binary is always matching the OS packages and libs needed --
//  and return the list of Infiniband devices on the system
func ParseDevices() ([]SimpleRDMADevice, error) {
	var devs []SimpleRDMADevice

	return devs, nil
}

//func getRdmaDeivces(masterNetDevice string) ([]Device, error) {
//	var devs []Device
//	// Get all RDMA device list
//	ibvDevList, err := ibverbs.IbvGetDeviceList()
//	if err != nil {
//		return nil, err
//	}
//
//	netDevList, err := GetVfNetDevice(masterNetDevice)
//	if err != nil {
//		return nil, err
//	}
//
//	for _, d := range ibvDevList {
//		for _, n := range netDevList {
//			dResource, err := getRdmaDeviceResource(d.Name)
//			if err != nil {
//				continue
//			}
//			nResource, err := getNetDeviceResource(n)
//			if err != nil {
//				continue
//			}
//
//			// the same device
//			if bytes.Compare(dResource, nResource) == 0 {
//				devs = append(devs, Device{
//					RdmaDevice: d,
//					NetDevice:  n,
//				})
//			}
//		}
//	}
//	return devs, nil
//}

func getRdmaDeviceResource(name string) ([]byte, error) {
	resourceFile := fmt.Sprintf(RdmaDeviceResource, name)
	data, err := ioutil.ReadFile(resourceFile)
	return data, err
}

func getNetDeviceResource(name string) ([]byte, error) {
	resourceFile := fmt.Sprintf(NetDeviceResource, name)
	data, err := ioutil.ReadFile(resourceFile)
	return data, err
}

func deviceExists(devs []*pluginapi.Device, id string) bool {
	for _, d := range devs {
		if d.ID == id {
			return true
		}
	}
	return false
}

func watchXIDs(ctx context.Context, devs []*pluginapi.Device, xids chan<- *pluginapi.Device) {
	for {
		select {
		case <-ctx.Done():
			return
		}

		// TODO: check RDMA device healthy status
	}
}
