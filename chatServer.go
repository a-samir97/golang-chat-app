package main

type WsServer struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

// to create a new WsServer type
func NewWebSocketServer() *WsServer {
	return &WsServer{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
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
		case message := <- server.broadcast:
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
	server.clients[client] = true
}

func (server *WsServer) unregisterClient(client *Client) {
	if _, ok := server.clients[client]; ok {
		// delete client from clients array
		delete(server.clients, client)
	}
}
