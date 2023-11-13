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

	ch.buf.WriteInt32(int32(handshake.Length()))

	handshake.Write(&ch.buf)

	err := ch.Send()
	if err != nil {
		return err
	}

	ch.buf.Reset()

	// Receives length + request_id + status_code
	err = ch.receive(4 + 8 + 4)
	if err != nil {
		return err
	}

	ch.buf.Reset()

	respLen := ch.buf.ReadInt32()
	reqId := ch.buf.ReadInt64()
	errorCode := ch.buf.ReadInt32()

	if errorCode != 0 {
		return errors.New(fmt.Sprintf("erorr=%d, request_id=%d", errorCode, reqId))
	}

	ch.buf.Reset()

	err = ch.receive(int(respLen))
	if err != nil {
		return err
	}

	ch.buf.Reset()

	resp := ch.buf.ReadHandshakeResponse()

	if resp.Code == 0 {
		return nil
	}

	return errors.New(resp.Message)
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
