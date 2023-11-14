package serdes

import (
	"errors"
	"fmt"
	"net"
)

type Channel struct {
	conn net.Conn

	buf IgniteBuffer
}

func (ch Channel) Close() error {
	return ch.conn.Close()
}

func (ch Channel) Send() error {
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

	err := ch.Send()
	if err != nil {
		return err
	}

	ch.buf.Reset()

	// Receives length + request_id + status_code
	err = ch.receive(IntBytes)
	if err != nil {
		return err
	}

	ch.buf.Reset()

	respLength := ch.buf.ReadInt32()

	err = ch.receive(int(respLength))
	if err != nil {
		return err
	}

	ch.buf.Position(IntBytes)

	success := ch.buf.ReadBoolean()

	if !success {
		resp := ch.buf.ReadHandshakeFailResponse()

		return errors.New(fmt.Sprintf("error[code=%d,msg=%s,major=%d,minor%d,maint=%d]",
			resp.Code, resp.Message, resp.Major, resp.Minor, resp.Maintenance))
	}

	resp := ch.buf.ReadHandshakeResponse()

	fmt.Println(fmt.Sprintf("serverNodeId=%s", resp.NodeId))

	return nil
}

func CreateChannel(addr string) (*Channel, error) {
	if len(addr) == 0 {
		return nil, errors.New("address is empty")
	}

	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	ch := Channel{conn, CreateIgniteBuffer()}

	err = ch.handshake()
	if err != nil {
		return nil, err
	}

	return &ch, nil
}
