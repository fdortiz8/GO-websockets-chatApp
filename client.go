// client.go will handle anything pertaining to a single client
package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

var (
	pongWait = 10 * time.Second // how long to wait for pong response
	
	pingInterval = (pongWait * 9) / 10 // frequency at which we send pings through connection
	// ping has to be lower than pong wait. pingInterval should be 90% of pongwait
)

type ClientList map[*Client]bool // client list using a map and use bool to see if they are there

type Client struct {
	connection 	*websocket.Conn 		// each client must have a websocket connectino
	manager 	*Manager 				// pointer to manager that is managing the client
	
	chatroom string
	
	// egress is used to avoid concurrent writes on the websocket connection
	egress chan Event
}


// factory for the client. makes newClients
// each new client accepts a websocket connection and a Manager who manages it 
// returns a pointer to a Client
func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client {
		connection: conn,
		manager: 	manager,
		egress: make(chan Event),
	}
}

func (c *Client) readMessages() {
	
	// if connection is closed then remove the client
	defer func() {
		// clean up connection
		c.manager.removeClient(c)
	} ()
	
	// wait and read for pong returned from client
	if err := c.connection.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Println(err)
		return
	}
	
	c.connection.SetReadLimit(512) // limit how much data can be sent in a message. for security and prevent malicious intent
	// limiting and setting the readLimit is called fixing the jumbo frames
	
	c.connection.SetPongHandler(c.pongHandler) // handle pong response from client and reset time out timer
	
	for { // continous open connection for messages	
		// returns error if connection is closed
		_, payload, err := c.connection.ReadMessage()
		
		if err != nil {
			
			// check if closed connection was normal
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}
		
		var request Event
		
		// marshal is converting go objects into json or vise versa to unmarshal
		if err := json.Unmarshal(payload, &request); err != nil {
			log.Printf("error marshalling event :%v", err)
			break
		}
		
		if err := c.manager.routeEvent(request, c); err != nil {
			log.Println("error handeling message: ", err)
		}
	}
}

func (c *Client) writeMessages() {
	defer func() {
		c.manager.removeClient(c)
	} ()
	
	ticker := time.NewTicker(pingInterval)
	
	for {
		// do not write directly into the websocket. only one message allowed at a time. 
		// write to egress channel that will allow us to ensure that only one message is being sent to
		// the websocket at a time. therefore allowing us to ensure concurrency
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Println("connection closed: ", err)
				}
				return 
			}
			
			data, err := json.Marshal(message)
			if err != nil {
				log.Println(err)
				return
			}
			
			if err := c.connection.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			log.Println("message sent")
			
		case <-ticker.C:
			log.Println("ping")
			
			// send a ping to the Client
			if err := c.connection.WriteMessage(websocket.PingMessage, []byte(``)); err != nil {
				log.Println("write msg err: ", err)
				return
			}
		}
	}
}

func (c *Client) pongHandler(pongMsg string) error {
	log.Println("pong")
	return c.connection.SetReadDeadline(time.Now().Add(pongWait)) // hand pong is received you MUST reset the timer in order for connection not to time out
	// using ping pong to check if connection is still alive is called "checking the heartbeat of the connection"
}