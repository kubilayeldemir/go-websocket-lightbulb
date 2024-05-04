package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func main() {
	http.HandleFunc("/lightbulb", lightBulbHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var clients = make(map[*websocket.Conn]bool)
var clientsMutex sync.Mutex
var lightBulb = false
var lightBulbMutex sync.Mutex

func lightBulbHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Connection Opened: ", r.RemoteAddr)
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade err:", err)
		return
	}
	defer func() {
		DeleteClientFromConnectionsWithLock(c)
		err := c.Close()
		if err != nil {
			return
		}
	}()

	AddClientToConnectionsWithLock(c)

	resMessage := GetLightBulbStateAsByteArrayWithLock()
	if err := c.WriteMessage(websocket.TextMessage, resMessage); err != nil {
		fmt.Println("Error writing message:", err)
	}

	for {
		_, _, err := c.ReadMessage()
		log.Println("Light bulb switched: ", r.RemoteAddr)
		if err != nil {
			log.Println("read:", err)
			break
		}

		resMessage := SwitchLightBulbAndGetStateWithLock()
		BroadcastNewLightBulbStateToAllClients(resMessage)
	}
}

func BroadcastNewLightBulbStateToAllClients(resMessage []byte) {
	clientsMutex.Lock()
	for client := range clients {
		go BroadcastNewLightBulbStateToClient(resMessage, client)
	}
	clientsMutex.Unlock()
}

func BroadcastNewLightBulbStateToClient(resMessage []byte, client *websocket.Conn) {
	if err := client.WriteMessage(websocket.TextMessage, resMessage); err != nil {
		fmt.Println("Error writing message:", err)
	}
}

func DeleteClientFromConnectionsWithLock(c *websocket.Conn) {
	clientsMutex.Lock()
	delete(clients, c)
	clientsMutex.Unlock()
}

func AddClientToConnectionsWithLock(c *websocket.Conn) {
	clientsMutex.Lock()
	clients[c] = true
	clientsMutex.Unlock()
}

func GetLightbulbStateAsByteArray() []byte {
	var resMessage []byte

	if lightBulb {
		resMessage = []byte("true")
	} else {
		resMessage = []byte("false")
	}
	return resMessage
}

func GetLightBulbStateAsByteArrayWithLock() []byte {
	defer lightBulbMutex.Unlock()
	lightBulbMutex.Lock()
	return GetLightbulbStateAsByteArray()
}

func SwitchLightBulb() {
	lightBulb = !lightBulb
}

func SwitchLightBulbAndGetStateWithLock() []byte {
	defer lightBulbMutex.Unlock()
	lightBulbMutex.Lock()
	SwitchLightBulb()
	return GetLightbulbStateAsByteArray()
}
