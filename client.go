package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Max wait time when writing message to peer
	writeWait = 10 * time.Second

	// Max time till next pong from peer
	pongWait = 60 * time.Second

	// Send ping interval, must be less then pong wait time
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 10000
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// The upgrader is used to upgrade the HTTP server connection to the WebSocket protocol.
//the return value of this function is a WebSocket connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	// actual websocket connection
	ID       uuid.UUID `json:"id"`
	conn     *websocket.Conn
	wsServer *WsServer
	send     chan []byte
	rooms    map[*Room]bool
	Name     string `json:"name"`
}

func newClient(conn *websocket.Conn, wsServer *WsServer, name string) *Client {
	// new client in the websocket connection
	return &Client{
		conn:     conn,
		wsServer: wsServer,
		send:     make(chan []byte, 256),
		rooms:    make(map[*Room]bool),
		Name:     name,
		ID:       uuid.New(),
	}
}

func (client *Client) GetName() string {
	return client.Name
}

// handle websocket requests from clients requests
func ServeWs(wsServer *WsServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	name, ok := r.URL.Query()["name"]

	if err != nil {
		log.Println(err)
		return
	}
	if !ok || len(name[0]) < 1 {
		log.Println("URL Param name is missing")
		return
	}

	// new client connects to the websocket
	client := newClient(conn, wsServer, name[0])

	go client.writePump()
	go client.readPump()

	wsServer.register <- client

	fmt.Println("New Client joined the hub")
	fmt.Println(client.Name)
	fmt.Println(client.rooms)
}

//In the readPump Goroutine,
//the client will read new messages send over the WebSocket connection.
//It will do so in an endless loop until the client is disconnected.
func (client *Client) readPump() {
	defer func() {
		client.disconnect()
	}()

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	// start endless loop, waiting for messages from client

	for {
		_, jsonMessage, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("unexpected close error : %v", err)
			}
			break
		}

		client.handleNewMessage(jsonMessage)
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// the Websocket server closed the channel
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				log.Println("Closed the channel, in writing...")
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(client.send)

			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		//The writePump is also responsible for keeping the connection alive
		//by sending ping messages to the client with the interval given in pingPeriod.
		//If the client does not respond with a pong, the connection is closed.
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		}
	}
}

func (client *Client) disconnect() {
	client.wsServer.unregister <- client
	for room := range client.rooms {
		room.unregister <- client
	}
	close(client.send)
	client.conn.Close()
}

// Handle join a new client to room
func (client *Client) handleJoinRoomMessage(message Message) {
	roomName := message.Message

	client.joinRoom(roomName, nil)
}

// Handle leave user from room
func (client *Client) handleLeaveRoomMessage(message Message) {
	room := client.wsServer.findRoomByID(message.Message)

	if room == nil {
		return
	}

	if _, ok := client.rooms[room]; ok {
		delete(client.rooms, room)
	}
	room.unregister <- client
}

// Handle New Message
func (client *Client) handleNewMessage(jsonMessage []byte) {
	var message Message
	// if there is any error while deseralizing the json string to
	// message object
	if err := json.Unmarshal(jsonMessage, &message); err != nil {
		log.Printf("Error on unmarshal JSON message %s", err)
	}

	// attach the client object as the sender of the message
	message.Sender = client

	switch message.Action {
	case SendMessageAction:
		// send messages to specific room now,
		// which room is the mesasge target here
		roomID := message.Target.GetId()
		// use this name to find the room
		if room := client.wsServer.findRoomByID(roomID); room != nil {
			room.broadcast <- &message
		}

	case JoinRoomAction:
		client.handleJoinRoomMessage(message)
	case LeaveRoomAction:
		client.handleLeaveRoomMessage(message)
	case JoinPrivateRoomAction:
		client.handleJoinPrivateRoom(message)
	}
}

// joining private room
// combine IDs of the client and target
func (client *Client) handleJoinPrivateRoom(message Message) {
	target := client.wsServer.findClienyByID(message.Message)
	// if there is no client, will not make the private room
	if target == nil {
		return
	}

	// create a unique room name combined the 2 IDs
	roomName := message.Message + client.ID.String()

	client.joinRoom(roomName, target)
	target.joinRoom(roomName, client)
}

// joining a room for both private and public
func (client *Client) joinRoom(roomName string, sender *Client) {
	room := client.wsServer.findRoomByName(roomName)
	if room == nil {
		room = client.wsServer.createRoom(roomName, sender != nil)
	}

	// don't allow to join private rooms through public room message
	if sender == nil && room.Private {
		return
	}

	if !client.InRoom(room) {
		client.rooms[room] = true
		room.register <- client
		client.notifyRoomJoined(room, sender)
	}
}

// check if the client in the room or not
func (client *Client) InRoom(room *Room) bool {
	if _, ok := client.rooms[room]; ok {
		return true
	}
	return false
}

// notify the client of the new room he joined
func (client *Client) notifyRoomJoined(room *Room, sender *Client) {
	message := Message{
		Action: RoomJoinedAction,
		Target: room,
		Sender: sender,
	}
	client.send <- message.encode()
}
