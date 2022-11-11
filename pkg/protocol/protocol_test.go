package protocol

import (
	"reflect"
	"testing"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
	"github.ibm.com/mbg-agent/cmd/gateway/state"

)

func TestFrame(t *testing.T) {
	hostService := service.Service{Id: "iperf3Client.Lon", Ip: "127.0.0.1:5001", Domain: "Inner", Policy: "Forward"}
	destService := service.Service{Id: "iperf3Server.Australia", Ip: "127.0.0.1:5001", Domain: "Inner", Policy: "Forward"}
	mbgID       := "mbg.Lon"
	Frame := createFrame(hostService, destService, mbgID)
	buf := convFrame2Buf(Frame)
	frame := Buf2ControlFrame(buf)

	if !reflect.DeepEqual(destService, frame.destService) {
		t.Errorf("FAILED: expected %v, got %v\n", destService, frame.destService)
	}

	if !reflect.DeepEqual(hostService, frame.hostService) {
		t.Errorf("FAILED: expected %v, got %v\n", hostService, frame.hostService)
	}

}
