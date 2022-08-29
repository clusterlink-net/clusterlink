package setupFrame

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	service "github.ibm.com/ei-agent/pkg/serviceMap"
)

type setupFrameS struct {
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

/********************* SetupFrame Functions- clien side *****************/
func SendFrame(cl, sn net.Conn, destIp, destPort string) error {
	//destIp, destPort := netutils.GetConnIp(cl)
	s := service.GetService(destPort)

	//create frame
	sFrame := createFrame(s, destIp, destPort)
	setupFrameBuf := convFrame2Buf(sFrame)

	_, err := sn.Write(setupFrameBuf)
	if err != nil {
		fmt.Printf("[sendSetupFrame]: Write error %v\n", err)
		return err
	}
	return nil
}

func createFrame(s service.Service, destIp string, destPort string) setupFrameS {
	return setupFrameS{Service: s, DestIp: destIp, DestPort: destPort}
}

//Conver setup frame to buffer for sending through connection
func convFrame2Buf(sFrame setupFrameS) []byte {

	setupFrameBuf := make([]byte, maxSetupBufferSize)

	byteOrder.PutUint32(setupFrameBuf[BufSizePos:BufSizePos+BufSizeSize], uint32(maxSetupBufferSize))
	byteOrder.PutUint32(setupFrameBuf[ServicePos:ServicePos+ServiceSize], sFrame.Service.Id)

	destIpB := net.ParseIP(sFrame.DestIp)
	copy(setupFrameBuf[DestIpPos:DestIpPos+len(destIpB)], destIpB)

	destPortB := []byte(sFrame.DestPort)
	copy(setupFrameBuf[DestPortPos:DestPortPos+DestPortSize], destPortB)

	//fmt.Println("[SendFrame]", setupFrameBuf)
	return setupFrameBuf
}

/********************* SetupFrame Functions- server side *****************/
func GetSetupPacket(cl net.Conn) setupFrameS {
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
	sFrame.Print()
	return sFrame
}

func convBuf2Frame(sFrameBuf []byte) setupFrameS {
	var sFrame setupFrameS
	sFrame.Service.Id = byteOrder.Uint32(sFrameBuf[ServicePos : ServicePos+ServiceSize])
	sFrame.Service.Name = service.ConvertId2Name(sFrame.Service.Id)
	sFrame.DestIp = net.IP(sFrameBuf[DestIpPos+DestIpSize-ipv4ByteSize : DestIpPos+DestIpSize]).String()
	sFrame.DestPort = string(sFrameBuf[DestPortPos : DestPortPos+DestPortSize])
	return sFrame
}

func GetServiceIp(packet []byte) string {

	ip := net.IP(packet[DestIpPos+DestIpSize-ipv4ByteSize : DestIpPos+DestIpSize]) //Getting just ip addr
	ipS := ip.String()
	return ipS
}

func (s *setupFrameS) Print() {
	println("setup Frame- service id:", s.Service.Id, ", service name:", s.Service.Name, ", Destination ip:", s.DestIp, ",Destination port:", s.DestPort)

}
