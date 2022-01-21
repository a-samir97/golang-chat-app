package main

type WsServer struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	rooms      map[*Room]bool
}

// to create a new WsServer type
func NewWebSocketServer() *WsServer {
	return &WsServer{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		rooms:      make(map[*Room]bool),
	}
}

// run our server, acccepting requests
// The Run() function will run infinitely and listens to the channels.
// One’s new requests present themself, they will be handled through dedicated functions
func (server *WsServer) Run() {
	for {
		select {
		case client := <-server.register:
			server.registerClient(client)
		case client := <-server.unregister:
			server.unregisterClient(client)
		case message := <-server.broadcast:
			server.broadcastToClients(message)
		}
	}
}

func (server *WsServer) broadcastToClients(message []byte) {
	for client := range server.clients {
		client.send <- message
	}
}
func (server *WsServer) registerClient(client *Client) {
	// register a new clients
	// append the new clients to clients array with true
	server.notifyClientJoined(client)
	server.listOnlineClients(client)
	server.clients[client] = true
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		// delete client from clients array
		delete(server.clients, client)
		server.notifyClientLeft(client)
	}
}

// find room by name
func (server *WsServer) findRoomByName(name string) *Room {
	var foundRoom *Room
	for room := range server.rooms {
		if room.name == name {
			foundRoom = room
			break
		}
	}
	return foundRoom
}

// create name by passing name
func (server *WsServer) createRoom(name string) *Room {
	room := NewRoom(name)
	go room.RunRoom()
	server.rooms[room] = true
	return room
}

// to notify existing users when a new one joins
func (server *WsServer) notifyClientJoined(client *Client) {
	message := &Message{
		Action: UserJoinredAction,
		Sender: client,
	}

	server.broadcastToClients(message.encode())
}

// notify existing users when one leaves
func (server *WsServer) notifyClientLeft(client *Client) {
	message := &Message{
		Action: UserLeftAction,
		Sender: client,
	}

	server.broadcastToClients(message.encode())
}

// list all existing users to the joining user
// get all online users
func (server *WsServer) listOnlineClients(client *Client) {
	for existingClient := range server.clients {
		message := &Message{
			Action: UserJoinredAction,
			Sender: existingClient,
		}
		client.send <- message.encode()
	}
}
