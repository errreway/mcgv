package serdes

import (
	"errors"
	"fmt"
	"net"
)

type Channel struct {
	conn net.Conn

	buf IgniteBuffer

	idGen int64

	topVer    int64
	minTopVer int32
}

type IgniteRequest interface {
	Write(buf *IgniteBuffer)

	OpCode() int16

	ReadResponse(buf *IgniteBuffer) interface{}
}

const (
	ErrorFlag                   = 1
	AffinityTopologyChangedFlag = 1 << 1
	NotificationFlag            = 1 << 2
)

func (ch Channel) Send(req IgniteRequest) (interface{}, error) {
	ch.buf.Reset()
	ch.buf.WriteInt32(0) // Reserve space for actual length
	ch.buf.WriteInt16(req.OpCode())

	requestId := ch.requestId()

	ch.buf.WriteInt64(requestId)
	req.Write(&ch.buf)

	requestLen := ch.buf.idx - IntBytes

	ch.buf.Position(0)

	ch.buf.WriteInt32(int32(requestLen))
	ch.buf.Position(requestLen + IntBytes)

	err := ch.sendAll()
	if err != nil {
		return nil, err
	}

	ch.buf.Reset()

	err = ch.receive(IntBytes)
	if err != nil {
		return nil, err
	}

	ch.buf.Position(0)

	respLength := ch.buf.ReadInt32()

	err = ch.receive(int(respLength))
	if err != nil {
		return nil, err
	}

	ch.buf.Position(IntBytes)
	ch.buf.Limit(int(respLength + IntBytes))

	requestId0 := ch.buf.ReadInt64()

	if requestId != requestId0 {
		panic(fmt.Sprintf("different id[req_id=%d, resp_id=%d]", requestId, requestId0))
	}

	flags := ch.buf.ReadInt16()

	if checkFlag(flags, AffinityTopologyChangedFlag) {
		ch.topVer = ch.buf.ReadInt64()
		ch.minTopVer = ch.buf.ReadInt32()
	}

	if checkFlag(flags, ErrorFlag) {
		statusCode := ch.buf.ReadInt32()
		errorMessage := ch.buf.ReadString()
		return nil, errors.New(fmt.Sprintf("error[code=%d,msg=%s,req=%d]", statusCode, *errorMessage, requestId0))
	}

	return req.ReadResponse(&ch.buf), nil
}

func (ch Channel) Close() error {
	return ch.conn.Close()
}

func (ch Channel) sendAll() error {
	n, err := ch.conn.Write(ch.buf.buf[0:ch.buf.idx])
	if err != nil {
		return err
	}

	for n < ch.buf.idx {
		n0, err0 := ch.conn.Write(ch.buf.buf[n:ch.buf.idx])
		if err0 != nil {
			return err0
		}
		n += n0
	}

	return nil
}

func (ch Channel) receive(bytes int) error {
	start := ch.buf.idx
	finish := start + bytes

	n, err := ch.conn.Read(ch.buf.buf[start:finish])
	if err != nil {
		return err
	}

	for start+n < finish {
		start += n
		n, err = ch.conn.Read(ch.buf.buf[start:finish])
		if err != nil {
			return err
		}
	}

	return nil
}

func (ch Channel) handshake() error {
	handshake := CreateHandshakeRequest()

	handshake.Features = &[]byte{1}

	ch.buf.WriteInt32(-1) // Reserve space for actual length

	handshake.Write(&ch.buf)

	requestLen := ch.buf.idx - IntBytes

	ch.buf.Reset()

	ch.buf.WriteInt32(int32(requestLen))
	ch.buf.Position(requestLen + IntBytes)

	err := ch.sendAll()
	if err != nil {
		return err
	}

	ch.buf.Reset()

	err = ch.receive(IntBytes)
	if err != nil {
		return err
	}

	ch.buf.Position(0)

	respLength := ch.buf.ReadInt32()

	err = ch.receive(int(respLength))
	if err != nil {
		return err
	}

	ch.buf.Position(IntBytes)
	ch.buf.Limit(int(respLength + IntBytes))

	success := ch.buf.ReadBoolean()

	if !success {
		resp := handshake.ReadFailResponse(&ch.buf).(HandshakeFailResponse)

		return errors.New(fmt.Sprintf("error[code=%d,msg=%s,major=%d,minor%d,maint=%d]",
			resp.Code, *resp.Message, resp.Major, resp.Minor, resp.Maintenance))
	}

	resp := handshake.ReadResponse(&ch.buf).(HandshakeResponse)

	fmt.Println(fmt.Sprintf("serverNodeId=%s", resp.NodeId))

	return nil
}

func checkFlag(flags int16, flag int16) bool {
	return flags&flag != 0
}

func (ch Channel) requestId() int64 {
	res := ch.idGen
	ch.idGen += 1
	return res
}

func CreateChannel(addr string) (*Channel, error) {
	if len(addr) == 0 {
		return nil, errors.New("address is empty")
	}

	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	ch := Channel{conn, CreateIgniteBuffer(), 0, 0, 0}

	err = ch.handshake()
	if err != nil {
		return nil, err
	}

	return &ch, nil
}
