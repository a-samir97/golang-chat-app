package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

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
	conn     *websocket.Conn
	wsServer *WsServer
	send     chan []byte
}

func newClient(conn *websocket.Conn, wsServer *WsServer) *Client {
	// new client in the websocket connection
	return &Client{conn: conn, wsServer: wsServer, send: make(chan []byte, 256)}
}

// handle websocket requests from clients requests
func ServeWs(wsServer *WsServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}
	// new client connects to the websocket
	client := newClient(conn, wsServer)

	wsServer.register <- client
	fmt.Println("New Client joined the hub")
	fmt.Println(client)
	fmt.Printf("%p\n", &client)
	fmt.Println(wsServer.clients)
}

//In the readPump Goroutine,
//the client will read new messages send over the WebSocket connection.
//It will do so in an endless loop until the client is disconnected.
func (client *Client) readPump() {
	defer func() {
		client.conn.Close()
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

		client.wsServer.broadcast <- jsonMessage
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
