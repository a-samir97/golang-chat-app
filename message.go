package main

import (
	"encoding/json"
	"log"
)

const SendMessageAction = "send-message"
const JoinRoomAction = "join-room"
const LeaveRoomAction = "leave-room"
const UserJoinredAction = "user-join"
const UserLeftAction = "user-left"

type Message struct {
	// Action: what is the action of the message reqesuting
	// send message, join room or leave room
	Action string `json:"action"`

	// the actual message, could be the message you want to send,
	// or the name of the room for joining or leaving
	Message string `json:"message"`

	// target for the message, room for example or someone
	Target string `json:"target"`

	// who is sending this message ?
	Sender *Client `json:"sender"`
}

func (message *Message) encode() []byte {
	json, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
	}
	return json
}
