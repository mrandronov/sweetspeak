package websockets

import (
	"fmt"
	log "sweetspeak/logging"
	"sweetspeak/message"
	"sync"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

var (
        CloseErrors = []int{websocket.CloseNormalClosure, websocket.CloseAbnormalClosure}
)

type WebsocketHandler struct {
	sync.Mutex
	ReadCh  chan message.WSMessage
	WriteCh chan message.WSMessage
	conn    *websocket.Conn
	active  bool
}

func New() *WebsocketHandler {
	w := &WebsocketHandler{
		ReadCh:  make(chan message.WSMessage),
		WriteCh: make(chan message.WSMessage),
	}

	return w
}

func (w *WebsocketHandler) WithConn(conn *websocket.Conn) *WebsocketHandler {
	w.conn = conn
	return w
}

func (w *WebsocketHandler) Start() *WebsocketHandler {
	go w.ReadPump()
        w.active = true
	return w
}

func (w *WebsocketHandler) Connect(addr string) error {
	urlStr := fmt.Sprintf("ws://%s", addr)
	log.Debug("websockets: dialing %s", urlStr)
	conn, _, err := websocket.DefaultDialer.Dial(urlStr, nil)
	if err != nil {
		return err
	}

	w.conn = conn
	w.active = true

	return nil
}

func (w *WebsocketHandler) ReadPump() {
	for !w.IsClosed() {
		if err := w.read(); err != nil {
			log.Error("websockets: read: %v", err)
		}
	}
}

func (w *WebsocketHandler) read() error {
	_, msgBytes, err := w.conn.ReadMessage()
	if websocket.IsCloseError(err, CloseErrors...) { 
		log.Warn("websockets: connection closed")
		w.Close()
		return nil
	} else if err != nil {
		return err
	}

	var wsMsg message.WSMessage
	err = yaml.Unmarshal(msgBytes, &wsMsg)
	if err != nil {
		return err
	}

	w.ReadCh <- wsMsg

	return nil
}

func (w *WebsocketHandler) Read() message.WSMessage {
	return <-w.ReadCh
}

func (w *WebsocketHandler) Write(msg message.WSMessage) error {
	return w.conn.WriteMessage(websocket.BinaryMessage, []byte(msg.String()))
}

func (w *WebsocketHandler) IsClosed() bool {
	w.Lock()
	defer w.Unlock()

	return !w.active
}

func (w *WebsocketHandler) Close() {
	w.Lock()
	defer w.Unlock()

	w.active = false
	w.conn.Close()

	close(w.ReadCh)
	close(w.WriteCh)
}
