package client

import (
	"sweetspeak/chat"
	"sweetspeak/consts"
	log "sweetspeak/logging"
	"sweetspeak/message"
	"sweetspeak/user"
	"sweetspeak/websockets"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type (
	Client struct {
		sync.Mutex
		ID        string
		ws        *websockets.WebsocketHandler
		User      *user.User
		Connected bool
		Chat      *chat.Chat
	}
)

func NewDefault() *Client {
	c := &Client{
		ID: uuid.NewString(),
	}

	return c
}

func New(id string, usr *user.User, conn *websocket.Conn) *Client {
	var ws *websockets.WebsocketHandler
	ws = nil
	if conn != nil {
		ws = websockets.New().WithConn(conn)
	}

	c := &Client{
		ID:   id,
		ws:   ws,
		User: usr,
	}

	return c
}

func (c *Client) Start() error {
	log.Debug("attempt client connect")
	wsHandler := websockets.New()

	err := wsHandler.Connect(consts.Addr)
	if err != nil {
		return err
	}

	wsHandler.Start()

	// Send introduction message immediately on start
	introMsg := message.NewIntroductionMessage(
		c.ID,
		*c.User,
	)
	if err = wsHandler.Write(introMsg); err != nil {
		return err
	}

	log.Debug("introduction sent")

	c.ws = wsHandler
	c.Connected = true

        go c.ReadMessages()

	log.Debug("connected successfully")

	return nil
}

func (c *Client) ReadMessages() {
        for {
                c.readMessage()
        }
}

func (c *Client) readMessage() {
	c.Lock()
	defer c.Unlock()

	if !c.Connected {
		return
	}

	var wsMsg message.WSMessage
	select {
	case wsMsg = <-c.ws.ReadCh:
	default:
	}

        if wsMsg.MessageID == "" {
                return
        }

	log.Debug("got message: %s:%v", wsMsg.MessageID, wsMsg.MessageType)

        err := c.HandleMessage(wsMsg)
        if err != nil {
                log.Error("client: failed to handle message: %v", err)
        }
}

func (c *Client) HandleMessage(wsMsg message.WSMessage) error {
        switch wsMsg.MessageType {
        case message.ChatResponseMsg:
                cr, err := wsMsg.ToChatResponse()
                if err != nil {
                        return err
                }

                c.Chat = chat.New(cr.ChatID, "example chat", cr.Users)
                log.Debug("client: receive chat response, starting chat (%s)", cr.ChatID)
        case message.TextMsg:
                tm, err := wsMsg.ToTextMessage()
                if err != nil {
                        return err
                }

                c.Chat.AddMessage(tm)
                log.Debug("client: receive text message for chat (%s), content: %s", tm.ChatID, tm.Content)
        }

        return nil
}

func (c *Client) SendChatMessage(content string) {
	if !c.Connected {
                log.Warn("client: sendChatMessage: not connected")
		return
	}

        if c.Chat == nil {
                log.Warn("client: sendChatMessage: chat is nil")
                return
        }

	textMsg := message.NewTextMessage(c.Chat.ID, *c.User, content)

	err := c.ws.Write(textMsg)
	if err != nil {
		log.Error("client: send chat message: %v", err)
                return
	}

        log.Debug("client: chat message sent (content=%s)", content)
}

func (c *Client) SendChatRequest(to string) {
	chatReq := message.NewChatRequest(c.User.Name, to)

	err := c.ws.Write(chatReq)
	if err != nil {
		log.Error("client: send chat request: %v", err)
	}

        log.Debug("client: chat request sent")
}
