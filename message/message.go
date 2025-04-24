package message

import (
	"fmt"
	"time"

	"sweetspeak/user"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type MessageType int

const (
        NonMsg MessageType = iota
	TextMsg
	StatusMsg
	ChatRequestMsg
	ChatResponseMsg
	IntroductionMsg
)

type ChatStatus int

const (
	ChatOpenStatus ChatStatus = iota
	UsrNotFoundStatus
	NotConnected
)

type (
	WSMessage struct {
		MessageID   string      `yaml:"message_id"`
		MessageType MessageType `yaml:"message_type"`
		Payload     interface{} `yaml:"payload"`
	}

	IntroductionMessage struct {
		ClientID string    `yaml:"client_id"`
		User     user.User `yaml:"user_info"`
	}

	TextMessage struct {
		ChatID    string    `yaml:"chat_id"`
		From      user.User `yaml:"from"`
		Timestamp time.Time `yaml:"timestamp"`
		Content   string    `yaml:"content"`
	}

	StatusMessage struct {
		From      string    `yaml:"from"`
		To        string    `yaml:"to"`
		Timestamp time.Time `yaml:"timestamp"`
		Online    bool      `yaml:"online"`
	}

	ChatRequest struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	}

	ChatResponse struct {
		ChatID string      `yaml:"chat_id"`
		Users  []user.User `yaml:"users"`
		Status ChatStatus  `yaml:"chat_status"`
	}
)

func NewWSMessage(messageType MessageType, payload interface{}) WSMessage {
	return WSMessage{
		MessageID:   uuid.New().String(),
		MessageType: messageType,
		Payload:     payload,
	}
}

func (w *WSMessage) String() string {
	data, _ := yaml.Marshal(w)
	return string(data)
}

func (w *WSMessage) UnmarshalYAML(value *yaml.Node) error {
	var tmp struct {
		MessageID   string      `yaml:"message_id"`
		MessageType MessageType `yaml:"message_type"`
		Payload     yaml.Node   `yaml:"payload"`
	}

	if err := value.Decode(&tmp); err != nil {
		return err
	}

	w.MessageID = tmp.MessageID
	w.MessageType = tmp.MessageType

	switch w.MessageType {
	case IntroductionMsg:
		var data IntroductionMessage
		if err := tmp.Payload.Decode(&data); err != nil {
			return err
		}
		w.Payload = data
	case TextMsg:
		var data TextMessage
		if err := tmp.Payload.Decode(&data); err != nil {
			return err
		}
		w.Payload = data
	case ChatRequestMsg:
		var data ChatRequest
		if err := tmp.Payload.Decode(&data); err != nil {
			return err
		}
		w.Payload = data
	case ChatResponseMsg:
		var data ChatResponse
		if err := tmp.Payload.Decode(&data); err != nil {
			return err
		}
		w.Payload = data
	}

	return nil
}

func (w *WSMessage) ToIntroduction() (IntroductionMessage, error) {
	if im, ok := w.Payload.(IntroductionMessage); ok {
		return im, nil
	}

	return IntroductionMessage{}, fmt.Errorf("payload is not IntroductionMessage")
}

func (w *WSMessage) ToTextMessage() (TextMessage, error) {
	if tm, ok := w.Payload.(TextMessage); ok {
		return tm, nil
	}
	return TextMessage{}, fmt.Errorf("payload is not TextMessage")
}

func (w *WSMessage) ToStatusMessage() (StatusMessage, error) {
	if sm, ok := w.Payload.(StatusMessage); ok {
		return sm, nil
	}
	return StatusMessage{}, fmt.Errorf("payload is not StatusMessage")
}

func (w *WSMessage) ToChatRequest() (ChatRequest, error) {
	if cr, ok := w.Payload.(ChatRequest); ok {
		return cr, nil
	}
	return ChatRequest{}, fmt.Errorf("payload is not ChatRequest")
}

func (w *WSMessage) ToChatResponse() (ChatResponse, error) {
	if cr, ok := w.Payload.(ChatResponse); ok {
		return cr, nil
	}
	return ChatResponse{}, fmt.Errorf("payload is not ChatResponse")
}

func NewIntroductionMessage(clientID string, u user.User) WSMessage {
	return NewWSMessage(IntroductionMsg, IntroductionMessage{
		ClientID: clientID,
		User:     u,
	})
}

func NewTextMessage(chatID string, from user.User, content string) WSMessage {
	return NewWSMessage(TextMsg, TextMessage{
		ChatID:    chatID,
		From:      from,
		Timestamp: time.Now(),
		Content:   content,
	})
}

func NewChatRequest(from, to string) WSMessage {
	return NewWSMessage(ChatRequestMsg, ChatRequest{
		From: from,
		To:   to,
	})
}

func NewChatResponse(chatID string, users []user.User, status ChatStatus) WSMessage {
	return NewWSMessage(ChatResponseMsg, ChatResponse{
		ChatID: chatID,
		Users:  users,
		Status: status,
	})
}
