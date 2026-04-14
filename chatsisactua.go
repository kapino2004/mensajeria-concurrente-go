package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Configuración del Upgrader para permitir conexiones de prueba [cite: 2]
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub gestiona el mapa de clientes activos y el broadcast [cite: 47, 54]
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			fmt.Println("Conectado")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				fmt.Println("Desconectado")
			}
		case message := <-h.broadcast:
			// Envío sin bloqueo a todos los canales [cite: 46, 50]
			for client := range h.clients {
				if err := client.WriteMessage(websocket.TextMessage, message); err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
		}
	}
}

// Maneja la interacción WebSocket individual por cliente [cite: 39, 41]
func handleConnections(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrade: %v", err)
		return
	}
	hub.register <- conn

	// Goroutine por cliente para lectura concurrente 
	go func() {
		defer func() { hub.unregister <- conn }()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Aquí se integrará la persistencia asíncrona en BD más adelante [cite: 14, 30]
			fmt.Printf("Msg: %s\n", string(msg))
			hub.broadcast <- msg
		}
	}()
}

func main() {
	hub := newHub()
	go hub.run() // Ejecucion concurrente del Hub [cite: 50]

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleConnections(hub, w, r)
	})

	fmt.Println("Servidor en :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}