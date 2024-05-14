package connector

import (
	"crypto/tls"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tomas-qstarrs/nano/cluster"
	"github.com/tomas-qstarrs/nano/codec"
	"github.com/tomas-qstarrs/nano/codec/plaincodec"
	"github.com/tomas-qstarrs/nano/log"
	"github.com/tomas-qstarrs/nano/serializer"
	"github.com/tomas-qstarrs/nano/serializer/auto"

	"github.com/tomas-qstarrs/nano/message"
	"github.com/tomas-qstarrs/nano/packet"
)

type (

	// Callback represents the callback type which will be called
	// when the correspond events is occurred.
	Callback func(data interface{})

	// Connector is a tiny Nano client
	Connector struct {
		Options

		conn           net.Conn      // low-level connection
		die            chan struct{} // connector close channel
		chSend         chan []byte   // send queue
		mid            uint64        // message id
		sid            uint64        // session id
		connected      int32         // connected state 1: disconnected : 0
		connectedEvent Callback      // connected callback
		chReady        chan struct{} // connector ready channel

		// events handler
		muEvents        sync.RWMutex
		events          map[string]Callback // stores all events by key:event name value:callback
		unexpectedEvent Callback            // un registered event run this callback

		// response handler
		muResponses sync.RWMutex
		responses   map[uint64]Callback

		codecEntity codec.CodecEntity
	}
)

// NewConnector create a new Connector
func NewConnector(opts ...Option) *Connector {
	c := &Connector{
		Options: Options{
			Serializer: auto.Serializer,
		},
		die:             make(chan struct{}),
		chSend:          make(chan []byte, 256),
		mid:             1,
		sid:             0,
		connected:       0,
		connectedEvent:  func(data interface{}) {},
		chReady:         make(chan struct{}, 1),
		events:          map[string]Callback{},
		unexpectedEvent: func(data interface{}) {},
		responses:       map[uint64]Callback{},
	}

	for i := range opts {
		opt := opts[i]
		opt(&c.Options)
	}

	if c.Options.Logger != nil {
		log.Use(c.Logger)
	}

	if c.Options.Codec == nil {
		c.Codec = plaincodec.NewCodec()
	}
	c.codecEntity = c.Codec.Entity(c.Options.Dictionary)

	return c
}

// Start connects to the server and send/recv between the c/s
func (c *Connector) Start(addr string) (err error) {
	if c.IsWebSocket {
		c.conn, err = c.getWebSocketConn(addr)
	} else {
		c.conn, err = c.getConn(addr)
	}
	if err != nil {
		return err
	}

	go c.write()

	go c.read()

	atomic.StoreInt32(&c.connected, 1)
	go c.connectedEvent(nil)
	c.chReady <- struct{}{}

	return nil
}

func (c *Connector) getConn(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

func (c *Connector) getWebSocketConn(addr string) (net.Conn, error) {
	u := url.URL{Scheme: "ws", Host: addr, Path: c.WebSocketPath}
	dialer := websocket.DefaultDialer
	var conn *websocket.Conn
	var err error
	dialer.EnableCompression = c.WebSocketCompression
	conn, _, err = dialer.Dial(u.String(), nil)
	if err != nil {
		u.Scheme = "wss"
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		conn, _, err = dialer.Dial(u.String(), nil)
		if err != nil {
			return nil, err
		}
	}

	return cluster.NewWSConn(conn)
}

func (c *Connector) Ready() <-chan struct{} {
	return c.chReady
}

// Name returns the name for connector
func (c *Connector) Name() string {
	return c.name
}

// OnConnected set the callback which will be called when the client connected to the server
func (c *Connector) OnConnected(callback Callback) {
	c.connectedEvent = callback
}

// GetMid returns current message id
func (c *Connector) GetMid() uint64 {
	return c.mid
}

// Request send a request to server and register a callbck for the response
func (c *Connector) Request(route string, v interface{}, callback Callback) error {
	var data []byte
	switch v := v.(type) {
	case []byte:
		data = v
	default:
		var err error
		data, err = c.Serialize(v)
		if err != nil {
			return err
		}
	}

	msg := &message.Message{
		Type:      message.Request,
		Branch:    c.Branch,
		ShortVer:  c.ShortVersion,
		Route:     route,
		ID:        c.mid,
		UnixTime:  uint32(time.Now().Unix()),
		SessionID: atomic.LoadUint64(&c.sid),
		DataType:  uint32(serializer.ToType(c.Serializer)),
		Data:      data,
	}

	c.setResponseHandler(c.mid, callback)
	if err := c.sendMessage(msg); err != nil {
		c.setResponseHandler(c.mid, nil)
		return err
	}

	return nil
}

// Notify send a notification to server
func (c *Connector) Notify(route string, v interface{}) error {
	var data []byte
	switch v := v.(type) {
	case []byte:
		data = v
	default:
		var err error
		data, err = c.Serialize(v)
		if err != nil {
			return err
		}
	}

	msg := &message.Message{
		Type:      message.Notify,
		Branch:    c.Branch,
		ShortVer:  c.ShortVersion,
		Route:     route,
		ID:        0,
		UnixTime:  uint32(time.Now().Unix()),
		SessionID: atomic.LoadUint64(&c.sid),
		DataType:  uint32(serializer.ToType(c.Serializer)),
		Data:      data,
	}
	return c.sendMessage(msg)
}

// On add the callback for the event
func (c *Connector) On(event string, callback Callback) {
	c.muEvents.Lock()
	defer c.muEvents.Unlock()

	c.events[event] = callback
}

// OnUnexpectedEvent sets callback for events that are not "On"
func (c *Connector) OnUnexpectedEvent(callback Callback) {
	c.unexpectedEvent = callback
}

// Close closes the connection, and shutdown the benchmark
func (c *Connector) Close() {
	defer func() {
		_ = recover()
	}()
	c.conn.Close()
	close(c.die)
}

// Connected returns the status whether connector is conncected
func (c *Connector) Connected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// Serialize marshals customized data into byte slice
func (c *Connector) Serialize(v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	}

	data, err := c.Serializer.Marshal(v)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Deserialize Unmarshals byte slice into customized data
func (c *Connector) Deserialize(data []byte, v interface{}) error {
	var err error
	if err = c.Serializer.Unmarshal(data, v); err != nil {
		return err
	}

	return nil
}

func (c *Connector) Send(data []byte) {
	c.send(data)
}

func (c *Connector) eventHandler(event string) (Callback, bool) {
	c.muEvents.RLock()
	defer c.muEvents.RUnlock()

	cb, ok := c.events[event]
	return cb, ok
}

func (c *Connector) responseHandler(mid uint64) (Callback, bool) {
	c.muResponses.RLock()
	defer c.muResponses.RUnlock()

	cb, ok := c.responses[mid]
	return cb, ok
}

func (c *Connector) setResponseHandler(mid uint64, cb Callback) {
	c.muResponses.Lock()
	defer c.muResponses.Unlock()

	if cb == nil {
		delete(c.responses, mid)
	} else {
		c.responses[mid] = cb
	}
}

func (c *Connector) sendMessage(msg *message.Message) error {
	data, err := c.codecEntity.EncodeMessage(msg)
	if err != nil {
		return err
	}

	packets := []*packet.Packet{{Length: len(data), Data: data}}
	payload, err := c.codecEntity.EncodePacket(packets)
	if err != nil {
		return err
	}

	c.mid++
	c.send(payload)

	return nil
}

func (c *Connector) write() {
	defer close(c.chSend)

	for {
		select {
		case data := <-c.chSend:
			if _, err := c.conn.Write(data); err != nil {
				log.Errorln(err)
				c.Close()
			}

		case <-c.die:
			atomic.StoreInt32(&c.connected, 0)
			return
		}
	}
}

func (c *Connector) send(data []byte) {
	c.chSend <- data
}

func (c *Connector) read() {
	buf := make([]byte, 2048)

	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Infof("Read [%s], connector will be closed immediately", err.Error())
			}
			c.Close()
			return
		}

		packets, err := c.codecEntity.DecodePacket(buf[:n])
		if err != nil {
			log.Errorln(err)
			c.Close()
			return
		}

		for i := range packets {
			p := packets[i]
			c.processPacket(p)
		}
	}
}

func (c *Connector) processPacket(p *packet.Packet) {
	msg, err := c.codecEntity.DecodeMessage(p.Data)
	if err != nil {
		log.Errorln(err)
		return
	}
	c.processMessage(msg)
}

func (c *Connector) processMessage(msg *message.Message) {
	atomic.StoreUint64(&c.sid, msg.SessionID)

	switch msg.Type {
	case message.Push:
		cb, ok := c.eventHandler(msg.Route)
		if ok {
			cb(msg)
		} else {
			c.unexpectedEvent(msg)
		}

	case message.Response:
		cb, ok := c.responseHandler(msg.ID)
		if !ok {
			log.Errorln("response handler not found", msg.ID)
			return
		}

		cb(msg)
		c.setResponseHandler(msg.ID, nil)

	case message.Request, message.Notify:
		log.Errorln("unsuported message type", msg.Type)

	default:
		log.Errorln("unsuported message type", msg.Type)
	}
}
