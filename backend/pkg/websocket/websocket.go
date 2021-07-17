package websocket

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

var upgrader = websocket.Upgrader{
	WriteBufferSize: 1024,
	ReadBufferSize:  1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	ws, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Fatalln(err)
		return ws, err
	}
	return ws, nil
}

/*
func Reader(conn *websocket.Conn) {
	for {
		messageType, p, err := conn.ReadMessage()

		if err != nil {
			log.Fatalln(err)
			return
		}
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Fatalln(err)
			return
		}
	}
}

func Writer(conn *websocket.Conn) {
	for {
		fmt.Println("Sending ...")
		messageType, r, err := conn.NextReader()

		if err !=  nil {
			log.Fatalln(err)
			return
		}

		w, err := conn.NextWriter(messageType)

		if err != nil {
			log.Fatalln(err)
			return
		}

		if _, err := io.Copy(w, r); err != nil {
			log.Fatalln(err)
			return
		}
		if err := w.Close(); err != nil {
			log.Fatalln(err)
			return
		}
	}
}

*/
