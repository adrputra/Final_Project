package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Message represents a chat message structure
type Message struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	RoomID  string `json:"roomId"`
	Sender  string `json:"sender"`
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	clients   map[*websocket.Conn]map[string]bool
	broadcast chan Message
	mutex     sync.Mutex
}

func (wsh *WebSocketHandler) handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()

	roomID := r.URL.Query().Get("roomid")

	if _, ok := wsh.clients[conn]; !ok {
		wsh.clients[conn] = make(map[string]bool)
	}

	wsh.mutex.Lock()
	wsh.clients[conn][roomID] = true
	wsh.mutex.Unlock()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			wsh.mutex.Lock()
			delete(wsh.clients[conn], roomID)
			wsh.mutex.Unlock()
			break
		}

		wsh.broadcast <- msg
	}
}

func (wsh *WebSocketHandler) handleMessages() {
	for {
		msg := <-wsh.broadcast
		wsh.mutex.Lock()
		for client, rooms := range wsh.clients {
			if rooms[msg.RoomID] {
				err := client.WriteJSON(msg)
				if err != nil {
					log.Println(err)
					client.Close()
					delete(wsh.clients, client)
				}
			}
		}
		wsh.mutex.Unlock()
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	router := mux.NewRouter()

	wsh := &WebSocketHandler{
		clients:   make(map[*websocket.Conn]map[string]bool),
		broadcast: make(chan Message),
	}

	go wsh.handleMessages()

	router.HandleFunc("/ws", wsh.handleConnections)

	// Serve static files (if your frontend is in a separate directory)
	// router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server is running on :8080")
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
}
