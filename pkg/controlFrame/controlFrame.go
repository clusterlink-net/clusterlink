/**********************************************************/
/* Package controlFrame contain all functions and control
/* message data structure
/**********************************************************/
package controlFrame

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

type controlFrameS struct {
	Service  service.Service
	DestIp   string
	DestPort string
}

var (
	byteOrder          = binary.BigEndian
	maxSetupBufferSize = 1024

	BufSizePos  = 0
	BufSizeSize = 4

	ServicePos  = BufSizePos + BufSizeSize
	ServiceSize = 4

	DestIpPos  = ServicePos + ServiceSize
	DestIpSize = 16

	DestPortPos  = DestIpPos + DestIpSize
	DestPortSize = 4

	ipv4ByteSize = 4
)

/********************* SetupFrame Functions- client side *****************/
//Get control message fields and send TCP buffer
func SendFrame(cl, mbg net.Conn, destIp, destPort, serviceType string) error {
	//destIp, destPort := netutils.GetConnIp(cl)
	s := service.GetService(serviceType)
	//create frame
	sFrame := createFrame(s, destIp, destPort)
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
func createFrame(s service.Service, destIp string, destPort string) controlFrameS {
	return controlFrameS{Service: s, DestIp: destIp, DestPort: destPort}
}

//Convert control frame to buffer for sending through connection
func convFrame2Buf(sFrame controlFrameS) []byte {

	controlFrameBuf := make([]byte, maxSetupBufferSize)

	byteOrder.PutUint32(controlFrameBuf[BufSizePos:BufSizePos+BufSizeSize], uint32(maxSetupBufferSize))
	byteOrder.PutUint32(controlFrameBuf[ServicePos:ServicePos+ServiceSize], sFrame.Service.Id)

	destIpB := net.ParseIP(sFrame.DestIp)
	copy(controlFrameBuf[DestIpPos:DestIpPos+len(destIpB)], destIpB)

	destPortB := []byte(sFrame.DestPort)
	copy(controlFrameBuf[DestPortPos:DestPortPos+DestPortSize], destPortB)

	//fmt.Println("[SendFrame]", controlFrameBuf)
	return controlFrameBuf
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
	sFrame := convBuf2Frame(bufData)
	sFrame.Print("[GetSetupPacket]")
	return sFrame
}

//Convert Buffer to SetupFrame
func convBuf2Frame(sFrameBuf []byte) controlFrameS {
	var sFrame controlFrameS
	sFrame.Service.Id = byteOrder.Uint32(sFrameBuf[ServicePos : ServicePos+ServiceSize])
	sFrame.Service.Name = service.ConvertId2Name(sFrame.Service.Id)
	sFrame.Service.Ip = service.ConvertId2Ip(sFrame.Service.Id)
	sFrame.DestIp = net.IP(sFrameBuf[DestIpPos+DestIpSize-ipv4ByteSize : DestIpPos+DestIpSize]).String()
	sFrame.DestPort = string(sFrameBuf[DestPortPos : DestPortPos+DestPortSize])
	return sFrame
}

//Get IP address from buffer
func GetServiceIp(packet []byte) string {
	ip := net.IP(packet[DestIpPos+DestIpSize-ipv4ByteSize : DestIpPos+DestIpSize]) //Getting just ip addr
	ipS := ip.String()
	return ipS
}

//print function for controlFrame struct
func (s *controlFrameS) Print(str string) {
	println(str, "control Frame- service id:", s.Service.Id, ", service name:", s.Service.Name, ", Destination ip:", s.DestIp, ",Destination port:", s.DestPort)

}
