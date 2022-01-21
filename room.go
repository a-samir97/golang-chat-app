package main

import (
	"fmt"

	"github.com/google/uuid"
)

const welcomeMessage = "%s joined the room"

type Room struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Private    bool      `json:"private"`
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
}

// Creata new room
func NewRoom(name string, private bool) *Room {
	return &Room{
		ID:         uuid.New(),
		Name:       name,
		Private:    private,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
	}
}

// get room id
func (room *Room) GetId() string {
	return room.ID.String()
}

// get room name
func (room *Room) GetName() string {
	return room.Name
}

// Run room, accepting requests
func (room *Room) RunRoom() {
	for {
		select {
		case client := <-room.register:
			room.registerClient(client)
		case client := <-room.unregister:
			room.unregisterClient(client)
		case message := <-room.broadcast:
			room.broadcastToClients(message.encode())
		}
	}
}

// Register a new user to the room
func (room *Room) registerClient(client *Client) {
	// notify user
	if !room.Private {
		room.notifyClientJoined(client)
	}
	room.clients[client] = true
}

// Unregsiter existing user from the room
func (room *Room) unregisterClient(client *Client) {
	if _, ok := room.clients[client]; ok {
		delete(room.clients, client)
	}
}

// Broadcast messages to the clients in the room
func (room *Room) broadcastToClients(message []byte) {
	for client := range room.clients {
		client.send <- message
	}
}

// notify client joined
func (room *Room) notifyClientJoined(client *Client) {
	message := &Message{
		Action:  SendMessageAction,
		Target:  room,
		Message: fmt.Sprintf(welcomeMessage, client.GetName()),
	}

	room.broadcastToClients(message.encode())
}
