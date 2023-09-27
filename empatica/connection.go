package empatica

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"viveSyncBroker/persistence"
)

var (
	device string
)

func Setup(ph *persistence.Handler) {
	port, err := strconv.Atoi(os.Getenv("E4_SERVER_PORT"))
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	server := NewStreamingServer(os.Getenv("E4_SERVER_ADDRESS"), port)
	err = server.Connect()
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	err = ConnectToFirstE4(server)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	go func() {
		for {
			ds := <-server.DataStream
			dataSet, err := json.Marshal(ds)
			if err != nil {
				fmt.Printf("Error: %+v\n", err)
			}
			ph.AddEntry(device, dataSet)
		}
	}()
}

func ConnectToFirstE4(server *StreamingServer) error {
	devices, err := ListDevices(server)
	if err != nil {
		return err
	}
	if len(devices) < 1 {
		return fmt.Errorf("No Device Available\n")
	}
	device = devices[0]
	connection := server.ConnectDevice(device)
	if connection.Command != "device_connect" {
		return fmt.Errorf("Failed to conenct to device: %+v\n", connection)
	}
	server.SubscribeStreams(StreamGsr | StreamIbi | StreamTmp)
	return nil
}

func ListDevices(server *StreamingServer) ([]string, error) {
	devices := server.ListDevices()
	numDevices, err := strconv.Atoi(devices.Arguments[0])
	if err != nil {
		return nil, err
	}
	deviceList := make([]string, numDevices)
	for i := 0; i < numDevices; i++ {
		deviceList[i] = devices.Arguments[i+2]
	}
	return deviceList, nil
}
