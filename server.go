package main

import (
	"net"
	"os"
	"path"
	"time"

	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	defaultResourceName = "tencent.com/rdma"
	serverSock          = pluginapi.DevicePluginPath + "rdma.sock"
	knemSysfsName       = "/sys/class/misc/knem"
)

// RdmaDevicePlugin implements the Kubernetes device plugin API
type RdmaDevicePlugin struct {
	devs []*pluginapi.Device
	// ID => Device
	devices map[string]Device
	socket  string
	//masterNetDevice string

	stop   chan interface{}
	health chan *pluginapi.Device

	server *grpc.Server
}

// NewRdmaDevicePlugin returns an initialized RdmaDevicePlugin
//func NewRdmaDevicePlugin(master string) *RdmaDevicePlugin {
func NewRdmaDevicePlugin() *RdmaDevicePlugin {
	//devices, err := GetDevices(master)
	devices, err := GetDevices()
	if err != nil {
		log.Errorf("Error getting RDMA devices: %v", err)
		return nil
	}

	var devs []*pluginapi.Device
	devMap := make(map[string]Device)
	for _, device := range devices {
		id := device.RdmaDevice.Name
		devs = append(devs, &pluginapi.Device{
			ID:     id,
			Health: pluginapi.Healthy,
		})
		devMap[id] = device
	}

	return &RdmaDevicePlugin{
		//masterNetDevice: master,
		socket:  serverSock,
		devs:    devs,
		devices: devMap,
		stop:    make(chan interface{}),
		health:  make(chan *pluginapi.Device),
	}
}

func (m *RdmaDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// Start starts the gRPC server of the device plugin
func (m *RdmaDevicePlugin) Start() error {
	err := m.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", m.socket)
	if err != nil {
		return err
	}

	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(m.server, m)

	go m.server.Serve(sock)

	// Wait for server to start by launching a blocking connection
	conn, err := dial(m.socket, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	go m.healthcheck()

	return nil
}

// Stop stops the gRPC server
func (m *RdmaDevicePlugin) Stop() error {
	if m.server == nil {
		return nil
	}

	m.server.Stop()
	m.server = nil
	close(m.stop)

	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *RdmaDevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn, err := dial(kubeletEndpoint, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(m.socket),
		ResourceName: resourceName,
	}
	log.Debugf("Plugin API version: %v", pluginapi.Version)

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

// ListAndWatch lists devices and update that list according to the health status
func (m *RdmaDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})

	for {
		select {
		case <-m.stop:
			return nil
		case d := <-m.health:
			// FIXME: there is no way to recover from the Unhealthy state.
			d.Health = pluginapi.Unhealthy
			s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})
		}
	}
}

func (m *RdmaDevicePlugin) unhealthy(dev *pluginapi.Device) {
	m.health <- dev
}

// Allocate returns the list of devices to expose in the container
func (m *RdmaDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	devs := m.devs
	responses := pluginapi.AllocateResponse{}
	var devicesList []*pluginapi.DeviceSpec
	var knemDeviceName = "/dev/knem"

	for _, req := range r.ContainerRequests {
		response := pluginapi.ContainerAllocateResponse{}

		log.Debugf("Request IDs: %v", req.DevicesIDs)

		for _, id := range req.DevicesIDs {
			if !deviceExists(devs, id) {
				return nil, fmt.Errorf("invalid allocation request: unknown device: %s", id)
			}

			var devPath string
			if dev, ok := m.devices[id]; ok {
				// TODO: to function
				devPath = fmt.Sprintf("/dev/infiniband/%s", dev.RdmaDevice.DevName)
				log.Debugf("device path found: %v", devPath)
			} else {
				continue
			}

			ds := &pluginapi.DeviceSpec{
				ContainerPath: devPath,
				HostPath:      devPath,
				Permissions:   "rw",
			}
			devicesList = append(devicesList, ds)
		}
		log.Debugf("Devices list from DevicesIDs: %v", devicesList)

		// for /dev/infiniband/rdma_cm
		rdma_cm_paths := []string{
			"/dev/infiniband/rdma_cm",
		}
		for _, dev := range rdma_cm_paths {
			devicesList = append(devicesList, &pluginapi.DeviceSpec{
				ContainerPath: dev,
				HostPath:      dev,
				Permissions:   "rw",
			})
		}

		// MPI (Intel at least) also requires the use of /dev/knem, add if present
		if _, err := os.Stat(knemSysfsName); err == nil {
			// Add the device to the list to mount in the container
			devicesList = append(devicesList, &pluginapi.DeviceSpec{
				ContainerPath: knemDeviceName,
				HostPath:      knemDeviceName,
				Permissions:   "rw",
			})
		}
		log.Debugf("Devices list after manual additions: %v", devicesList)

		response.Devices = devicesList

		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}

	return &responses, nil
}

func (m *RdmaDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (m *RdmaDevicePlugin) cleanup() error {
	if err := os.Remove(m.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (m *RdmaDevicePlugin) healthcheck() {
	ctx, cancel := context.WithCancel(context.Background())

	var xids chan *pluginapi.Device
	xids = make(chan *pluginapi.Device)
	go watchXIDs(ctx, m.devs, xids)

	for {
		select {
		case <-m.stop:
			cancel()
			return
		case dev := <-xids:
			m.unhealthy(dev)
		}
	}
}

// Serve starts the gRPC server and register the device plugin to Kubelet
func (m *RdmaDevicePlugin) Serve(resourceName string) error {
	err := m.Start()
	if err != nil {
		log.Errorf("Could not start device plugin: %v", err)
		return err
	}
	log.Infof("Starting to serve on %s", m.socket)

	err = m.Register(pluginapi.KubeletSocket, resourceName)
	if err != nil {
		log.Errorf("Could not register device plugin: %v", err)
		m.Stop()
		return err
	}
	log.Infof("Registered device plugin with Kubelet")

	return nil
}
