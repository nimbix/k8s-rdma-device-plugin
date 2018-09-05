package main

import (
	"flag"
	"os"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

//var (
//	MasterNetDevice = ""
//)

func main() {
	// Parse command-line arguments
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	//flagMasterNetDev := flag.String("master", "", "Master ethernet network device for SRIOV, ex: eth1")
	flagLogLevel := flag.String("log-level", "info", "Define the logging level: error, info, debug")
	flagResourceName := flag.String("resource-name", defaultResourceName, "Define the default resource name: tencent.com/rdma")
	flag.Parse()

	switch *flagLogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	}

	//if *flagMasterNetDev != "" {
	//	MasterNetDevice = *flagMasterNetDev
	//}

	log.Println("Fetching devices")

	//devList, err := GetDevices(MasterNetDevice)
	devList, err := GetDevices()
	if err != nil {
		log.Errorf("Error getting IB devices: %v", err)
		select {}
	}
	if len(devList) == 0 {
		log.Println("No IB devices found")
		select {}
	}

	log.Debugf("RDMA device list: %v", devList)
	log.Println("Starting FS watcher")
	watcher, err := newFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		log.Println("Failed to created FS watcher for device plugin path: ", pluginapi.DevicePluginPath)
		os.Exit(1)
	}
	defer watcher.Close()

	log.Println("Starting OS watcher")
	sigs := newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	restart := true
	var devicePlugin *RdmaDevicePlugin

	// Run this main loop until some event, a signal, breaks out and exits
L:
	for {
		if restart {
			if devicePlugin != nil {
				devicePlugin.Stop()
			}

			devicePlugin = NewRdmaDevicePlugin()
			if err := devicePlugin.Serve(*flagResourceName); err != nil {
				log.Println("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
			} else {
				restart = false
			}
		}

		select {
		case event := <-watcher.Events:
			if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("inotify: %s created, restarting", pluginapi.KubeletSocket)
				restart = true
			}

		case err := <-watcher.Errors:
			log.Printf("inotify: %s", err)

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				log.Println("Received SIGHUP, restarting")
				restart = true
			default:
				log.Printf("Received signal \"%v\", shutting down", s)
				devicePlugin.Stop()
				break L
			}
		}
	}
}
