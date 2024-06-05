package empatica

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"
	"strconv"
	"strings"
	"time"
)

type StreamingData struct {
	Stream    StreamType `json:"stream"`
	Timestamp time.Time  `json:"timestamp"`
	Values    []float64  `json:"values"`
}

type ResponseData struct {
	Command   string
	Arguments []string
}

type StreamingServer struct {
	Addr       net.TCPAddr
	Conn       *net.TCPConn
	Recv       chan ResponseData
	DataStream chan StreamingData
	conbuf     *bufio.Reader
}

type StreamType uint

const (
	StreamAcc StreamType = 1 << 0
	StreamBvp StreamType = 1 << 1
	StreamGsr StreamType = 1 << 2
	StreamIbi StreamType = 1 << 3
	StreamTmp StreamType = 1 << 4
	StreamBat StreamType = 1 << 5
	StreamTag StreamType = 1 << 6
)

var streamTypeStrings = []string{
	"acc",
	"bvp",
	"gsr",
	"ibi",
	"tmp",
	"bat",
	"tag",
}

var streamResolveTypes = map[string]StreamType{
	"E4_Acc":         StreamAcc,
	"E4_Bvp":         StreamBvp,
	"E4_Gsr":         StreamGsr,
	"E4_Temperature": StreamTmp,
	"E4_Hr":          StreamIbi, // IBI = 1/HR
	"E4_Battery":     StreamBat,
	"E4_Tag":         StreamTag,
}

func (st StreamType) String() string {
	n := st
	for i := 0; i < 7; i++ {
		if n == 1 {
			return streamTypeStrings[i]
		}
		n >>= 1
		if n < 1 {
			return ""
		}
	}
	return ""
}

func (st StreamType) MarshalJSON() ([]byte, error) {
	return json.Marshal(st.String())
}

func (st StreamType) Strings() []string {
	result := make([]string, 0, bits.OnesCount(uint(st)))
	for i := 0; i < 7; i++ {
		if st&(1<<i) == 1<<i {
			result = append(result, streamTypeStrings[i])
		}
	}
	return result
}

func NewStreamingServer(address string, port int) *StreamingServer {
	return &StreamingServer{
		Addr: net.TCPAddr{
			IP:   net.ParseIP(address),
			Port: port,
		},
	}
}

func (s *StreamingServer) Connect() error {
	var err error
	s.Conn, err = net.DialTCP("tcp", nil, &s.Addr)
	if err != nil {
		return err
	}
	s.Recv = make(chan ResponseData)
	s.DataStream = make(chan StreamingData)
	go func() {
		for {
			s.conbuf = bufio.NewReader(s.Conn)
			str, err := s.conbuf.ReadString('\n')
			if err != nil {
				fmt.Printf("Error: %+v\n", err)
			}
			str = strings.Trim(str, "\r\n")
			//			fmt.Printf("-- %s --\n", str)
			if isResponse(str) {
				s.Recv <- parseResponse(str)
			} else {
				streamData := parseStream(str)
				if streamData != nil {
					s.DataStream <- *streamData
				}
			}
		}
	}()
	return nil
}

func (s *StreamingServer) Send(cmd string, args ...string) error {
	cmdPlain := command(cmd, args...)
	//	fmt.Printf(">> %s <<\n", cmdPlain)
	_, err := s.Conn.Write(cmdPlain)
	if err != nil {
		return err
	}
	return nil
}

func (s *StreamingServer) ListDevices() ResponseData {
	err := s.Send("device_list")
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	return <-s.Recv
}

func (s *StreamingServer) ConnectDevice(id string) ResponseData {
	err := s.Send("device_connect", id)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	return <-s.Recv
}

func (s *StreamingServer) SubscribeStreams(streams StreamType) {
	for _, stream := range streams.Strings() {
		err := s.Send("device_subscribe", stream, "ON")
		if err != nil {
			fmt.Printf("Error: %+v\n", err)
		}
		fmt.Printf("Subscribed: %+v\n", <-s.Recv)
	}
}

func isResponse(s string) bool {
	return strings.HasPrefix(s, "R ")
}

func parseResponse(cmd string) ResponseData {
	parts := strings.Split(cmd, " ")
	return ResponseData{
		Command:   parts[1],
		Arguments: parts[2:],
	}
}

func parseStream(cmd string) *StreamingData {
	parts := strings.Split(cmd, " ")
	st, found := streamResolveTypes[parts[0]]
	if !found {
		return nil
	}
	ts, err := strconv.ParseFloat(strings.Replace(parts[1], ",", ".", 1), 64)
	if err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	values := make([]float64, len(parts)-2)
	for i := 2; i < len(parts); i++ {
		values[i-2], err = strconv.ParseFloat(strings.Replace(parts[i], ",", ".", 1), 64)
		if err != nil {
			fmt.Printf("Error: %+v\n", err)
		}
	}
	return &StreamingData{
		Stream:    st,
		Timestamp: time.UnixMicro(int64(ts * 1000000)),
		Values:    values,
	}
}

func command(cmd string, args ...string) []byte {
	return []byte(fmt.Sprintf("%s %s\r\n", cmd, strings.Join(args, " ")))
}
