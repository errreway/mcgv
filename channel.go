package ignite

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net"
	"sbt.ru/ignite-go/ignite/internal/bitset"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ErrorFlag                   = 1
	AffinityTopologyChangedFlag = 1 << 1
	NotificationFlag            = 1 << 2
)

const (
	open int32 = iota
	closed
)

const (
	V1_0_0  = "1.0.0"
	V1_1_0  = "1.0.0"
	V1_2_0  = "1.2.0"
	V1_3_0  = "1.3.0"
	V1_4_0  = "1.4.0"
	V1_5_0  = "1.5.0"
	V1_6_0  = "1.6.0"
	V1_7_0  = "1.7.0"
	Default = V1_7_0
)

type ClientStatus = uint

const (
	Success ClientStatus = iota
	Failed
	InvalidOpCode
	InvalidNodeState      = 10
	FunctionalityDisabled = 100
	CacheDoesNotExists    = 1000
	CacheExists           = 1001
	CacheConfigInvalid    = 1002
	TooManyCursors        = 1010
	ResourceDoesNotExists = 1011
	SecurityViolation     = 1012
	TxLimitExceeded       = 1020
	TxNotFound            = 1021
	TooManyComputeTasks   = 1030
	AuthFailed
)

type ClientError struct {
	Message string
}

type ClientProtocolError struct {
	ClientError
}

type ClientAuthenticationError struct {
	ClientError
}

type ClientServerError struct {
	ClientError
	Code int
}

type VersionMismatchError struct {
	ClientError
	Version ProtocolVersion
}

func (err *ClientError) Error() string {
	return err.Message
}

func (err *ClientServerError) Error() string {
	return fmt.Sprintf("%d: %s", err.Code, err.Message)
}

type Channel struct {
	socket            net.Conn
	idGen             atomic.Int64
	protocolCtx       atomic.Value
	topVer            int64
	minTopVer         int32
	pendingCh         chan int64
	pendingRequests   sync.Map
	doneCh            chan struct{}
	serverId          uuid.UUID
	status            int32
	supportedVersions map[string]bool
	closeErr          error
	closeWg           sync.WaitGroup
	clientCfg         *ClientConfiguration
}

type pendingRequest struct {
	id           int64
	requestData  []byte
	responseData []byte
	err          error
	doneCh       chan struct{}
}

func (req *pendingRequest) responseLength() (int, bool) {
	if req.responseData != nil && len(req.responseData) >= intBytes {
		res := binary.LittleEndian.Uint32(req.responseData)
		return int(res), true
	}
	return 0, false
}

func (req *pendingRequest) responseId() (int64, bool) {
	return responseId(req.responseData)
}

func responseId(packet []byte) (int64, bool) {
	if packet != nil && len(packet) >= longBytes {
		res := binary.LittleEndian.Uint64(packet)
		return int64(res), true
	}
	return 0, false
}

func newRequest(id int64, opCode int16, requestWriter func(input BinaryWriter) error) (*pendingRequest, error) {
	reqInput := NewBinaryWriter(64)

	reqInput.WriteInt32(0)

	// Handshake request
	if id != -1 {
		reqInput.WriteInt16(opCode)
		reqInput.WriteInt64(id)
	}
	err := requestWriter(reqInput)
	if err != nil {
		return nil, err
	}

	currPosition := reqInput.Position()
	reqInput.SetPosition(0)
	reqInput.WriteInt32(currPosition - intBytes)
	reqInput.SetPosition(currPosition)

	return &pendingRequest{
		id:          id,
		requestData: reqInput.Data(),
		doneCh:      make(chan struct{}, 1),
	}, nil
}

func (ch *Channel) ProtocolContext() ProtocolContext {
	res := ch.protocolCtx.Load()
	if res == nil {
		return nil
	}
	return res.(*protocolContextImpl)
}

func (ch *Channel) Send(ctx context.Context, opCode int16, requestWriter func(output BinaryWriter) error, responseReader func(input BinaryReader, err error)) {
	ch.send(ctx, ch.requestId(), opCode, requestWriter, responseReader)
}

func (ch *Channel) send(ctx context.Context, id int64, opCode int16, requestWriter func(output BinaryWriter) error, responseReader func(input BinaryReader, err error)) {
	req, err := newRequest(id, opCode, requestWriter)
	if err != nil {
		responseReader(nil, err)
		return
	}

	var deadlineSet = false
	if ctx == nil {
		ctx = context.Background()
	}
	if _, deadlineSet = ctx.Deadline(); !deadlineSet {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 1*time.Second)
		defer func() {
			cancel()
		}()
	}

	reqId := req.id
	ch.pendingRequests.Store(reqId, req)
	ch.pendingCh <- reqId

	defer func() {
		ch.pendingRequests.Delete(reqId)
	}()

	for {
		select {
		case <-ctx.Done():
			responseReader(nil, errors.New("request cancelled"))
			return
		case <-ch.doneCh:
			if ch.closeErr != nil {
				err = fmt.Errorf("connection closed on error %w", ch.closeErr)
			} else {
				err = errors.New("connection closed")
			}
			responseReader(nil, err)
			return
		case <-req.doneCh:
			if req.err != nil {
				responseReader(nil, req.err)
				return
			}

			input := NewBinaryReader(req.responseData, 0)

			// process handshake
			if reqId == -1 {
				responseReader(input, nil)
				return
			}

			resId := input.ReadInt64()
			if resId != reqId {
				panic(fmt.Sprintf("reqId != resId %d, %d", reqId, resId))
			}

			flags := input.ReadInt16()

			if checkFlag(flags, AffinityTopologyChangedFlag) {
				ch.topVer = input.ReadInt64()
				ch.minTopVer = input.ReadInt32()
			}

			if checkFlag(flags, ErrorFlag) {
				statusCode := int(input.ReadInt32())
				var errMsg string
				if errMsg, err = unmarshalString(input, false); err != nil {
					err = fmt.Errorf("broken output from server: %w", err)
				}
				req.err = &ClientServerError{
					ClientError{
						Message: errMsg,
					},
					statusCode,
				}
			}

			responseReader(input, req.err)
			return
		}
	}
}

func (ch *Channel) beginClose(err error) {
	if !atomic.CompareAndSwapInt32(&ch.status, open, closed) {
		return
	}
	ch.closeErr = err
	close(ch.doneCh)
	_ = ch.socket.Close()
}

func (ch *Channel) Close(err error) {
	ch.beginClose(err)
	ch.closeWg.Wait()
}

func (ch *Channel) writeLoop() {
	defer func() {
		ch.closeWg.Done()
	}()
	for {
		select {
		case id := <-ch.pendingCh:
			req, ok := ch.pendingRequests.Load(id)
			if !ok {
				panic("invalid request id")
			}
			if err := ch.writeFully(req.(*pendingRequest).requestData); err != nil {
				req.(*pendingRequest).err = err
				ch.beginClose(err)
				return
			}
		case <-ch.doneCh:
			return
		}
	}
}

func (ch *Channel) writeFully(data []byte) error {
	total, err := ch.socket.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to socket: %w", err)
	}
	for total < len(data) {
		n := 0
		n, err = ch.socket.Write(data[total:])
		if err != nil {
			return fmt.Errorf("failed to write to socket: %w", err)
		}
		total += n
	}
	return nil
}

const (
	messageBufferSize = 128 * 1024
)

func (ch *Channel) readLoop() {
	var err error
	var n int
	defer func() {
		ch.closeWg.Done()
	}()

	buf := make([]byte, messageBufferSize)
	packetAcc := newPacketAccumulator()
	handshakeDone := false
	for {
		if err = ch.socket.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
			break
		}
		n, err = ch.socket.Read(buf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			if errors.Is(err, io.EOF) {
				continue
			}
			break
		}
		if n == 0 {
			continue
		}
		packetAcc.append(buf[:n])

		for {
			data := packetAcc.data()
			if data == nil {
				break
			}

			if !handshakeDone {
				if ch.ProtocolContext() != nil {
					handshakeDone = true
				}
			}

			var id int64
			if !handshakeDone {
				id = -1
			} else {
				var ok bool
				id, ok = responseId(data)
				if !ok {
					break
				}
			}

			val, ok := ch.pendingRequests.Load(id)
			if !ok {
				break
			}

			req, ok := val.(*pendingRequest)
			if !ok {
				panic("invalid data in pending requests")
			}
			req.responseData = data

			req.doneCh <- struct{}{}
			close(req.doneCh)
			break
		}
	}
	ch.beginClose(err)
}

type packetAccumulator struct {
	buf         *bytes.Buffer
	currentSize int32
}

func newPacketAccumulator() *packetAccumulator {
	return &packetAccumulator{
		buf:         bytes.NewBuffer(make([]byte, 0, messageBufferSize)),
		currentSize: 0,
	}
}

func (pa *packetAccumulator) append(buf []byte) {
	pa.buf.Write(buf)
}

func (pa *packetAccumulator) data() []byte {
	if !pa.processData() {
		return nil
	}
	// prepare result
	size := int(pa.currentSize + intBytes)
	result := make([]byte, size)
	copy(result, pa.buf.Next(size))

	// copy remained data, reinitialize buffer.
	remainDataSlice := pa.buf.Next(pa.buf.Len())
	remainData := make([]byte, len(remainDataSlice))
	copy(remainData, remainDataSlice)
	if pa.buf.Cap() > messageBufferSize {
		pa.buf = bytes.NewBuffer(make([]byte, 0, messageBufferSize))
	} else {
		pa.buf.Reset()
	}
	if len(remainData) > 0 {
		pa.buf.Write(remainData)
	}
	pa.currentSize = 0

	return result
}

func (pa *packetAccumulator) processData() bool {
	if pa.currentSize <= 0 {
		if pa.buf.Len() < intBytes {
			return false
		}
		size := int32(binary.LittleEndian.Uint32(pa.buf.Next(intBytes)))
		if size < byteBytes {
			panic("packet size is less than minimal message size")
		}
		pa.currentSize = size
	}
	if pa.currentSize > 0 {
		size := int(pa.currentSize)
		if pa.buf.Len() < size {
			return false
		}
		return true
	}
	return false
}

func (ch *Channel) handshake(ver ProtocolVersion) error {
	cliCtx := NewProtocolContext(ver, UserAttributesFeature)
	for {
		srvCtx, err := ch.handshakeRound(cliCtx, ch.clientCfg.user, ch.clientCfg.password, ch.clientCfg.attrs)
		if err != nil {
			return err
		}
		if srvCtx == nil {
			break
		}
		// Try to do handshake with server version.
		cliCtx = srvCtx
	}
	return nil
}

func (ch *Channel) handshakeRound(cliCtx ProtocolContext, user string, password string, attrs map[string]string) (ProtocolContext, error) {
	var err error = nil
	var srvCtx ProtocolContext = nil
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer func() {
		cancel()
	}()
	writer := func(bw BinaryWriter) error {
		bw.WriteInt8(1)
		cliCtx.Marshall(bw)
		if cliCtx.SupportsAttributeFeature(UserAttributesFeature) {
			if len(attrs) == 0 {
				bw.WriteNull()
			} else {
				bw.WriteInt8(mapType)
				bw.WriteInt32(int32(len(attrs)))
				bw.WriteInt8(hashMap)
				for k, v := range attrs {
					marshalString(bw, k)
					marshalString(bw, v)
				}
			}
		}
		if cliCtx.SupportsAuthorization() && len(user) > 0 {
			marshalString(bw, user)
			marshalString(bw, password)
		}
		return nil
	}
	reader := func(input BinaryReader, err0 error) {
		if err0 != nil {
			err = err0
			return
		}
		success := input.ReadBool()
		if success {
			if cliCtx.SupportsBitmapFeatures() {
				var bitMaskBytes []byte = nil
				if bitMaskBytes, err = unmarshalBytes(input, false); err != nil {
					err = fmt.Errorf("broken output from server: %w", err)
					return
				}
				if bitMaskBytes != nil {
					cliCtx.UpdateAttributeFeatures(bitset.FromBytes(bitMaskBytes))
				}
			}
			if cliCtx.SupportsPartitionAwareness() {
				var serverId uuid.UUID
				if serverId, err = unmarshalUuid(input, false); err != nil {
					err = fmt.Errorf("broken output from server: %w", err)
					return
				}
				ch.serverId = serverId
			}
			ch.protocolCtx.Store(cliCtx)
		} else {
			srvCtx = NewProtocolContext(
				ProtocolVersion{Major: input.ReadInt16(), Minor: input.ReadInt16(), Patch: input.ReadInt16()},
			)
			var errMsg string
			if errMsg, err = unmarshalString(input, false); err != nil {
				err = fmt.Errorf("broken output from server: %w", err)
				return
			}
			errCode := Failed
			if input.Available() > 0 {
				errCode = uint(input.ReadUInt32())
			}
			cliVersion := cliCtx.Version()
			if errCode == AuthFailed {
				err = &ClientAuthenticationError{
					ClientError{errMsg},
				}
			} else if cliVersion.Compare(srvCtx.Version()) == 0 {
				err = &ClientProtocolError{
					ClientError{errMsg},
				}
			} else if exists := ch.supportedVersions[cliVersion.String()]; !exists || (!cliCtx.SupportsAuthorization() && len(user) > 0) {
				errMsg = fmt.Sprintf("Protocol version mismatch: client %v / server %v. Server details: %s",
					cliVersion,
					srvCtx.Version(),
					errMsg,
				)
				err = &ClientProtocolError{
					ClientError{errMsg},
				}
			}
		}
	}
	ch.send(ctx, -1, 0, writer, reader)
	if err != nil {
		return nil, err
	}
	return srvCtx, nil
}

func checkFlag(flags int16, flag int16) bool {
	return flags&flag != 0
}

func (ch *Channel) requestId() int64 {
	return ch.idGen.Add(1)
}

func CreateChannel(cfg *ClientConfiguration) (*Channel, error) {
	if cfg.addressesSupplier == nil {
		return nil, errors.New("address supplier is nil")
	}
	addresses, err := cfg.addressesSupplier()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain addresses: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errors.New("addresses are empty")
	}
	conn, err := net.Dial("tcp", addresses[0])
	if err != nil {
		return nil, err
	}
	if cfg.tlsConfigSupplier != nil {
		tlsCfg, err := cfg.tlsConfigSupplier()
		if err != nil {
			return nil, fmt.Errorf("failed to obtain tls config: %w", err)
		}

		tlsCon := tls.Client(conn, tlsCfg)
		if err = tlsCon.Handshake(); err != nil {
			return nil, fmt.Errorf("tls handshake failed: %w", err)
		}
		conn = tlsCon
	}
	ch := Channel{
		socket:            conn,
		pendingCh:         make(chan int64, 1024),
		doneCh:            make(chan struct{}),
		supportedVersions: make(map[string]bool),
		clientCfg:         cfg,
	}
	ch.supportedVersions[V1_0_0] = true
	ch.supportedVersions[V1_1_0] = true
	ch.supportedVersions[V1_2_0] = true
	ch.supportedVersions[V1_3_0] = true
	ch.supportedVersions[V1_4_0] = true
	ch.supportedVersions[V1_5_0] = true
	ch.supportedVersions[V1_6_0] = true
	ch.supportedVersions[V1_7_0] = true
	ch.closeWg.Add(1)
	go ch.writeLoop()
	ch.closeWg.Add(1)
	go ch.readLoop()
	ver, _ := ParseVersion(Default)
	err = ch.handshake(ver)
	if err != nil {
		return nil, err
	}
	return &ch, nil
}
