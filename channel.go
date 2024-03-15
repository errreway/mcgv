package ignite

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gitverse.ru/sbertech/ignite-go-client/internal/bitset"
	"net"
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

//go:generate go run golang.org/x/tools/cmd/stringer -type=ErrorCode
type ErrorCode uint

const (
	Success               ErrorCode = 0
	Failed                ErrorCode = 1
	InvalidOpCode         ErrorCode = 2
	InvalidNodeState      ErrorCode = 10
	FunctionalityDisabled ErrorCode = 100
	CacheDoesNotExists    ErrorCode = 1000
	CacheExists           ErrorCode = 1001
	CacheConfigInvalid    ErrorCode = 1002
	TooManyCursors        ErrorCode = 1010
	ResourceDoesNotExists ErrorCode = 1011
	SecurityViolation     ErrorCode = 1012
	TxLimitExceeded       ErrorCode = 1020
	TxNotFound            ErrorCode = 1021
	TooManyComputeTasks   ErrorCode = 1030
	AuthFailed            ErrorCode = 2000
)

type ClientError struct {
	Message string
}

type ClientConnectionError struct {
	ClientError
	err error
}

type ClientProtocolError struct {
	ClientError
}

type ClientAuthenticationError struct {
	ClientError
}

type ClientServerError struct {
	ClientError
	Code ErrorCode
}

func (err *ClientError) Error() string {
	return err.Message
}

func (err *ClientServerError) Error() string {
	return fmt.Sprintf("%s: %s", err.Code, err.Message)
}

func (err *ClientConnectionError) Error() string {
	msg := err.Message
	if len(msg) == 0 {
		msg = "connection failed"
	}
	if err.err != nil {
		return fmt.Sprintf("%s: %s", msg, err.err)
	}
	return msg
}

func createClientConnectionError(msg string, err error) *ClientConnectionError {
	return &ClientConnectionError{ClientError{msg}, err}
}

type Channel interface {
	Send(ctx context.Context, opCode int16, requestWriter func(output BinaryWriter) error, responseReader func(input BinaryReader, err error))
	ProtocolContext() ProtocolContext
	Close()
}

type tcpChannel struct {
	addr              string
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
	closeErr          atomic.Value
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

func responseId(packet []byte) (int64, bool) {
	if len(packet) >= longBytes {
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
		doneCh:      make(chan struct{}),
	}, nil
}

func (ch *tcpChannel) ProtocolContext() ProtocolContext {
	res := ch.protocolCtx.Load()
	if res == nil {
		return nil
	}
	return res.(*protocolContextImpl)
}

func (ch *tcpChannel) Send(ctx context.Context, opCode int16, requestWriter func(output BinaryWriter) error, responseReader func(input BinaryReader, err error)) {
	ch.send(ctx, ch.requestId(), opCode, requestWriter, responseReader)
}

func (ch *tcpChannel) send(ctx context.Context, id int64, opCode int16, requestWriter func(output BinaryWriter) error, responseReader func(input BinaryReader, err error)) {
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
		ctx, cancel = context.WithTimeout(ctx, ch.clientCfg.requestTimeout)
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
			responseReader(nil, ch.processCloseError("request cancelled"))
			return
		case <-ch.doneCh:
			responseReader(nil, ch.processCloseError("connection closed"))
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
				errMsg, err = unmarshalString(input, false)
				if err != nil {
					req.err = createClientConnectionError("broken output from server", err)
				} else {
					req.err = &ClientServerError{
						ClientError{
							Message: errMsg,
						},
						ErrorCode(statusCode),
					}
				}
			}
			responseReader(input, req.err)
			return
		}
	}
}

func (ch *tcpChannel) processCloseError(defaultMsg string) error {
	closeErr := ch.closeErr.Load()
	if closeErr == nil {
		return &ClientError{defaultMsg}
	}
	return &ClientConnectionError{ClientError{"connection error"}, closeErr.(error)}
}

func (ch *tcpChannel) beginClose(err error) {
	if !atomic.CompareAndSwapInt32(&ch.status, open, closed) {
		return
	}
	if err != nil {
		ch.closeErr.Store(err)
	}
	close(ch.doneCh)
	_ = ch.socket.Close()
}

func (ch *tcpChannel) Close() {
	ch.beginClose(nil)
	ch.closeWg.Wait()
}

func (ch *tcpChannel) writeLoop() {
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
				ch.beginClose(err)
				return
			}
		case <-ch.doneCh:
			return
		}
	}
}

func (ch *tcpChannel) writeFully(data []byte) error {
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

func (ch *tcpChannel) readLoop() {
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
			break
		}
		if n <= 0 {
			continue
		}
		packetAcc.append(buf[:n])
		data := packetAcc.data()
		if data == nil {
			continue
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
				err = createClientConnectionError("failed to parse response id", nil)
				break
			}
		}
		val, ok := ch.pendingRequests.Load(id)
		if !ok {
			err = createClientConnectionError(fmt.Sprintf("failed to load request with id %d", id), nil)
			break
		}
		req, ok := val.(*pendingRequest)
		if !ok {
			err = createClientConnectionError("invalid data in pending requests", nil)
			break
		}
		req.responseData = data
		close(req.doneCh)
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
	// read pa.currentSize since buffer is on position 4
	copy(result, pa.buf.Next(int(pa.currentSize)))
	// copy remained data, reinitialize buffer
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
		return pa.buf.Len() >= size
	}
	return false
}

func (ch *tcpChannel) handshake(ver ProtocolVersion) error {
	cliProtoCtx := NewProtocolContext(ver, UserAttributesFeature)
	ctx, cancel := context.WithTimeout(context.Background(), ch.clientCfg.requestTimeout*3)
	defer func() {
		cancel()
	}()
	for {
		srvCtx, err := ch.handshakeRound(ctx, cliProtoCtx, ch.clientCfg)
		if err != nil {
			return err
		}
		if srvCtx == nil {
			break
		}
		// Try to do handshake with server version.
		cliProtoCtx = srvCtx
	}
	return nil
}

func (ch *tcpChannel) handshakeRound(ctx context.Context, cliProtoCtx ProtocolContext, cliCfg *ClientConfiguration) (ProtocolContext, error) {
	var err error = nil
	var srvProtoCtx ProtocolContext = nil
	writer := func(bw BinaryWriter) error {
		bw.WriteInt8(1)
		cliProtoCtx.Marshall(bw)
		if cliProtoCtx.SupportsAttributeFeature(UserAttributesFeature) {
			if len(cliCfg.attrs) == 0 {
				bw.WriteNull()
			} else {
				bw.WriteInt8(mapType)
				bw.WriteInt32(int32(len(cliCfg.attrs)))
				bw.WriteInt8(hashMap)
				for k, v := range cliCfg.attrs {
					marshalString(bw, k)
					marshalString(bw, v)
				}
			}
		}
		if cliProtoCtx.SupportsAuthorization() && len(cliCfg.user) > 0 {
			marshalString(bw, cliCfg.user)
			marshalString(bw, cliCfg.password)
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
			if cliProtoCtx.SupportsBitmapFeatures() {
				var bitMaskBytes []byte = nil
				if bitMaskBytes, err = unmarshalBytes(input, false); err != nil {
					err = fmt.Errorf("broken output from server: %w", err)
					return
				}
				if bitMaskBytes != nil {
					cliProtoCtx.UpdateAttributeFeatures(bitset.FromBytes(bitMaskBytes))
				}
			}
			if cliProtoCtx.SupportsPartitionAwareness() {
				var serverId uuid.UUID
				if serverId, err = unmarshalUuid(input, false); err != nil {
					err = fmt.Errorf("broken output from server: %w", err)
					return
				}
				ch.serverId = serverId
			}
			ch.protocolCtx.Store(cliProtoCtx)
		} else {
			srvProtoCtx = NewProtocolContext(
				ProtocolVersion{Major: input.ReadInt16(), Minor: input.ReadInt16(), Patch: input.ReadInt16()},
			)
			var errMsg string
			if errMsg, err = unmarshalString(input, false); err != nil {
				err = fmt.Errorf("broken output from server: %w", err)
				return
			}
			errCode := Failed
			if input.Available() > 0 {
				errCode = ErrorCode(input.ReadUInt32())
			}
			cliVersion := cliProtoCtx.Version()
			if errCode == AuthFailed {
				err = &ClientAuthenticationError{
					ClientError{errMsg},
				}
			} else if cliVersion.Compare(srvProtoCtx.Version()) == 0 {
				err = &ClientProtocolError{
					ClientError{errMsg},
				}
			} else if exists := ch.supportedVersions[cliVersion.String()]; !exists || (!cliProtoCtx.SupportsAuthorization() && len(cliCfg.user) > 0) {
				errMsg = fmt.Sprintf("Protocol version mismatch: client %v / server %v. Server details: %s",
					cliVersion,
					srvProtoCtx.Version(),
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
	return srvProtoCtx, nil
}

func checkFlag(flags int16, flag int16) bool {
	return flags&flag != 0
}

func (ch *tcpChannel) requestId() int64 {
	return ch.idGen.Add(1)
}

func createTcpChannel(addr string, cfg *ClientConfiguration) (*tcpChannel, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, createClientConnectionError(fmt.Sprintf("failed to connect to %s", addr), err)
	}
	if cfg.tlsConfigSupplier != nil {
		tlsCfg, err := cfg.tlsConfigSupplier()
		if err != nil {
			return nil, createClientConnectionError("failed to obtain tls config", err)
		}
		tlsCon := tls.Client(conn, tlsCfg)
		if err = tlsCon.Handshake(); err != nil {
			return nil, createClientConnectionError("tls handshake failed", err)
		}
		conn = tlsCon
	}
	ch := tcpChannel{
		addr:              addr,
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
	ch.idGen.Store(1)
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
