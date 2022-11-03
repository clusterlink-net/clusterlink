package protocol

import (
	"reflect"
	"testing"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

func TestFrame(t *testing.T) {
	hostService := service.Service{Name: "iperf3", Id: "Lon", Ip: "127.0.0.1:5001", Domain: "Inner", Policy: "Forward"}
	destService := service.Service{Name: "iperf3", Id: "Australia", Ip: "127.0.0.1:5001", Domain: "Inner", Policy: "Forward"}
	Frame := createFrame(hostService, destService)
	buf := convFrame2Buf(Frame)
	frame := Buf2ControlFrame(buf)

	if !reflect.DeepEqual(destService, frame.destService) {
		t.Errorf("FAILED: expected %v, got %v\n", destService, frame.destService)
	}

	if !reflect.DeepEqual(hostService, frame.hostService) {
		t.Errorf("FAILED: expected %v, got %v\n", hostService, frame.hostService)
	}

}
