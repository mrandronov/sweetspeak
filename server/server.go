package server

import (
	"fmt"
	"net/http"
	"sweetspeak/chat"
	"sweetspeak/consts"
	log "sweetspeak/logging"
	"sweetspeak/message"
	"sweetspeak/user"
	"sweetspeak/websockets"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	IntroductionTimeout = 5 * time.Second
)

type (
	Server struct {
		sync.Mutex
		Clients   map[*ServerClient]interface{}
		Chats     map[string]*chat.Chat
		MessageCh chan serverMsg
	}

	ServerClient struct {
		ClientID  string
		User      user.User
		WSHandler *websockets.WebsocketHandler
		Connected bool
	}

	serverMsg struct {
		client *ServerClient
		msg    message.WSMessage
	}
)

func New() *Server {
	s := &Server{
		Clients:   make(map[*ServerClient]interface{}),
		Chats:     make(map[string]*chat.Chat),
		MessageCh: make(chan serverMsg, 1000),
	}

	return s
}

func (s *Server) Start() {
	log.Info("starting server...")

	go s.HandleClientMessages()
	go s.HandleClientDisconnect()

	http.HandleFunc("/", s.HandleWS)

	log.Info("listening on %s...", consts.Addr)
	if err := http.ListenAndServe(consts.Addr, nil); err != nil {
		panic(err)
	}
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("websocket upgrade: %v", err)
		return
	}

	// Listen for the introduction message so that
	// we can identify this client.

	ws := websockets.New().WithConn(c).Start()
	newClient := &ServerClient{
		WSHandler: ws,
	}

	wsMsg, ok := newClient.ReadWithTimeout(IntroductionTimeout)
	if !ok {
		log.Error("client introduction never received (%s)", c.RemoteAddr().String())
		return
	}

	if wsMsg.MessageType != message.IntroductionMsg {
		log.Error("client sent non-introduction (%s)", c.RemoteAddr().String())
		return
	}

	im, err := wsMsg.ToIntroduction()
	if err != nil {
		log.Error("reading client introduction: %v", err)
		return
	}

	newClient.ClientID = im.ClientID
	newClient.User = im.User
	newClient.Connected = true

	s.AddClient(newClient)

	log.Info("client connect success: %s %s", newClient.ClientID, newClient.User.Name)
}

func (s *Server) AddClient(client *ServerClient) {
	s.Lock()
	defer s.Unlock()

	s.Clients[client] = true
	go s.clientHandler(client)
}

func (s *Server) clientHandler(client *ServerClient) {
        log.Debug("client handler started for %s:%s", client.User.Name, client.ClientID)
	for {
		if client.WSHandler.IsClosed() {
			break
		}

		wsMsg, ok := client.Read()
		if !ok {
			continue
		}

                s.MessageCh <- serverMsg{client: client, msg: wsMsg}
	}

	client.Connected = false
        log.Warn("client disconnected (%s)", client.User.Name)
}

func (s *Server) HandleClientMessages() {
	log.Info("listening for client messages...")
	for msg := range s.MessageCh {
                s.Lock()
                client := msg.client
                wsMsg := msg.msg

		log.Debug("message received: %v", wsMsg.MessageType)

		if err := s.HandleMsg(client, wsMsg); err != nil {
			log.Error("reading client message: %v", err)
		}
                s.Unlock()
	}
}

func (s *Server) HandleClientDisconnect() {
	for {
		s.handleClientDisconnect()
	}
}

func (s *Server) handleClientDisconnect() {
	s.Lock()
	defer s.Unlock()

	for c := range s.Clients {
		if !c.Connected {
			delete(s.Clients, c)
                        log.Warn("client deleted from map (%s)", c.User.Name)
		}
	}

}

func (sc *ServerClient) Read() (message.WSMessage, bool) {
	var wsMsg message.WSMessage

	select {
	case wsMsg = <-sc.ReadCh():
	default:
		return message.WSMessage{}, false
	}

	return wsMsg, true
}

func (sc *ServerClient) ReadWithTimeout(timeout time.Duration) (message.WSMessage, bool) {
	var wsMsg message.WSMessage

	select {
	case wsMsg = <-sc.ReadCh():
	case <-time.NewTimer(timeout).C:
		return message.WSMessage{}, false
	}

	return wsMsg, true

}

func (sc *ServerClient) ReadCh() chan message.WSMessage {
	return sc.WSHandler.ReadCh
}

func (sc *ServerClient) Send(wsMsg message.WSMessage) error {
	return sc.WSHandler.Write(wsMsg)
}

func (s *Server) HandleMsg(client *ServerClient, wsMsg message.WSMessage) error {
	switch wsMsg.MessageType {
	case message.ChatRequestMsg:
		cr, err := wsMsg.ToChatRequest()
		if err != nil {
			return err
		}

		return s.RcvChatRequest(client, cr)
	case message.TextMsg:
		tm, err := wsMsg.ToTextMessage()
		if err != nil {
			return err
		}

		return s.RcvTextMessage(client, tm)
	default:
	}

	return nil
}

func (s *Server) RcvChatRequest(fromClient *ServerClient, chatRequest message.ChatRequest) error {
	toUser := chatRequest.To

	// Look up toUser first.
	toClient := s.LookupClient("", toUser)
	if toClient == nil {
		// Send user not found message.
		chatResp := message.NewChatResponse("", nil, message.UsrNotFoundStatus)
		fromClient.WSHandler.WriteCh <- chatResp
		return fmt.Errorf("client [%s] chat request: user not found (%v)", fromClient.ClientID, toUser)
	}

	// toClient is valid, create a local chat entry and
	// send an open response to both clients.

	var (
		chatID = uuid.NewString()
		users  = []user.User{
			fromClient.User,
			toClient.User,
		}
		chatResp = message.NewChatResponse(
			chatID,
			users,
			message.ChatOpenStatus,
		)
	)

	s.Chats[chatID] = chat.New(
		chatID,
		fmt.Sprintf("%s and %s's Chat", chatRequest.From, chatRequest.To),
		users,
	)

	log.Debug("chat request: sending chat response to users")

	if err := toClient.Send(chatResp); err != nil {
		return fmt.Errorf("chat request: to-client write: %v", err)
	}

	if err := fromClient.Send(chatResp); err != nil {
		return fmt.Errorf("chat request: from-client write: %v", err)
	}

	log.Info("chat request: successfully sent: %s", chatID)

	return nil
}

// Assume caller calls Lock()
func (s *Server) LookupClient(clientID string, userName string) *ServerClient {
	if clientID != "" {
		for c := range s.Clients {
			if c.ClientID == clientID {
				return c
			}
		}
	}

	if userName != "" {
		for c := range s.Clients {
			if c.User.Name == userName {
				return c
			}
		}
	}

	return nil
}

func (s *Server) LookupChat(chatID string) *chat.Chat {
	if c, ok := s.Chats[chatID]; ok {
		return c
	}
	return nil
}

func (s *Server) RcvTextMessage(fromClient *ServerClient, textMessage message.TextMessage) error {
        log.Debug("received text message: %s - %s", textMessage.ChatID, textMessage.Content)
	// Look up the chat
	clientChat := s.LookupChat(textMessage.ChatID)
	if clientChat == nil {
		// A Chat should have been established at this point.
		// If the chat cannot be found, something is really broken.

		// We should probably let fromClient know that something is not working.
		return fmt.Errorf("client [%s] text message: chat not found (%v)", fromClient.ClientID, textMessage.ChatID)
	}

	var (
		chatID = textMessage.ChatID
		wsMsg  = message.NewWSMessage(message.TextMsg, textMessage)
	)

	// Forward textMessage to all users in the chat.
	for _, u := range clientChat.Users {
		toClient := s.LookupClient("", u.Name)
		if toClient == nil {
			return fmt.Errorf("chat [%s]: client is nil (%s)", chatID, toClient.String())
		}

		if err := toClient.Send(wsMsg); err != nil {
			return fmt.Errorf("chat [%s]: client write (%s): %v", chatID, toClient.String(), err)
		}
	}

        log.Debug("message forwarded successfully for chat (%s)", clientChat.ID)

	return nil
}

func (sc *ServerClient) String() string {
	return fmt.Sprintf("%s:%s:%t", sc.ClientID, sc.User.Name, sc.Connected)
}
