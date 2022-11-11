/**********************************************************/
/* Package controlFrame contain all functions and control
/* message data structure
/**********************************************************/
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type controlFrameS struct {
	hostService service.Service
	destService service.Service
	//DestIp   string
	//DestPort string
}

var (
	byteOrder          = binary.BigEndian
	maxSetupBufferSize = 1024

	BufSizePos  = 0
	BufSizeSize = 4

	hostServicePos  = BufSizePos + BufSizeSize
	hostServiceSize = 256

	destServicePos  = hostServicePos + hostServiceSize
	destServiceSize = 256

	IdPos      = 0
	IdSize     = 32
	IpPos      = IdPos + IdSize
	IpSize     = 16
	PortPos    = IpPos + IpSize
	PortSize   = 4
	DomainPos  = PortPos + PortSize
	DomainSize = 16
	PolicyPos  = DomainPos + DomainSize
	PolicySize = 64

	ipv4ByteSize = 4
)

/********************* SetupFrame Functions- client side *****************/
//Get control message fields and send TCP buffer
func SendFrame(cl, mbg net.Conn, hostService service.Service, destServiceName string) error {
	//destIp, destPort := netutils.GetConnIp(cl)
	destService := service.GetService(destServiceName)
	//create frame
	sFrame := createFrame(hostService, destService)
	sFrame.Print("[sendSetupFrame]")
	controlFrameBuf := convFrame2Buf(sFrame)

	_, err := mbg.Write(controlFrameBuf)
	if err != nil {
		fmt.Printf("[sendSetupFrame]: Write error %v\n", err)
		return err
	}
	return nil
}

//convert control field to controlframe struct
func createFrame(hostService, destService service.Service) controlFrameS {
	return controlFrameS{hostService: hostService, destService: destService}
}

//Convert control frame to buffer for sending through connection
func convFrame2Buf(sFrame controlFrameS) []byte {

	controlFrameBuf := make([]byte, maxSetupBufferSize)

	hostBuf := service2Buf(sFrame.hostService)
	copy(controlFrameBuf[hostServicePos:hostServicePos+hostServiceSize], hostBuf)
	destBuf := service2Buf(sFrame.destService)
	copy(controlFrameBuf[destServicePos:destServicePos+destServiceSize], destBuf)

	//fmt.Println("[SendFrame]", controlFrameBuf)
	return controlFrameBuf
}

func service2Buf(s service.Service) []byte {
	serviceBuf := make([]byte, maxSetupBufferSize)
	copy(serviceBuf[IdPos:IdPos+IdSize], []byte(s.Id))
	ip := strings.Split(s.Ip, ":")[0]
	port := strings.Split(s.Ip, ":")[1]
	ipB := net.ParseIP(ip)
	copy(serviceBuf[IpPos:IpPos+len(ipB)], ipB)
	copy(serviceBuf[PortPos:PortPos+PortSize], []byte(port))
	copy(serviceBuf[DomainPos:DomainPos+DomainSize], []byte(s.Domain))
	copy(serviceBuf[PolicyPos:PolicyPos+PolicySize], []byte(s.Policy))
	return serviceBuf
}

/********************* SetupFrame Functions- server side *****************/
//listen to control message and return controlFrame struct
func GetSetupPacket(cl net.Conn) controlFrameS {
	bufData := make([]byte, maxSetupBufferSize)
	bufReadSize := 0
	for bufReadSize < maxSetupBufferSize {
		numBytes, err := cl.Read(bufData[bufReadSize:maxSetupBufferSize])
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				fmt.Printf("[clientToServer]: Read error %v\n", err)
			}
		}
		bufReadSize += numBytes
	}
	sFrame := Buf2ControlFrame(bufData)
	sFrame.Print("[GetSetupPacket]")
	return sFrame
}

//Convert Buffer to SetupFrame
func Buf2ControlFrame(sFrameBuf []byte) controlFrameS {
	var sFrame controlFrameS
	sFrame.hostService = Buf2Service(sFrameBuf[hostServicePos : hostServicePos+hostServiceSize])
	sFrame.destService = Buf2Service(sFrameBuf[destServicePos : destServicePos+destServiceSize])
	return sFrame
}

func Buf2Service(sBuf []byte) service.Service {
	var s service.Service

	s.Id = buf2String(sBuf[IdPos : IdPos+IdSize])
	s.Ip = net.IP(sBuf[IpPos+IpSize-ipv4ByteSize : IpPos+IpSize]).String()
	s.Ip = s.Ip + ":" + buf2String(sBuf[PortPos:PortPos+PortSize])
	s.Domain = buf2String(sBuf[DomainPos : DomainPos+DomainSize])
	s.Policy = buf2String(sBuf[PolicyPos : PolicyPos+PolicySize])

	return s
}

//Get IP address from buffer
// func GetServiceIp(packet []byte) string {
// 	ip := net.IP(packet[DestIpPos+DestIpSize-ipv4ByteSize : DestIpPos+DestIpSize]) //Getting just ip addr
// 	ipS := ip.String()
// 	return ipS
// }

//print function for controlFrame struct
func (s *controlFrameS) Print(str string) {
	fmt.Printf("%v control Frame- host service : %v, dest service name: %v", str, s.hostService, s.destService)

}

func buf2String(buf []byte) string {
	return strings.Replace(string(buf), "\x00", "", -1)
}
