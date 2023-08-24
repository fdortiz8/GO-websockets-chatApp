// https://www.youtube.com/watch?v=pKpKv9MKN-E&ab_channel=ProgrammingPercy
// manager.go will be used to manage anything that has to do with the websocket
package main

// takes http request and upgrades it to a websocket connection instead of regular http request
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
) 

var (
	websocketUpgrader = websocket.Upgrader{
		CheckOrigin: checkOrigin,
		ReadBufferSize: 1024,
		WriteBufferSize: 1024,
	}
)

type Manager struct {
	clients ClientList 
	sync.RWMutex 		// since we can have many people concurrently connecting to the API protect it with a Read/Write Mutex
	
	otps RetentionMap 
	
	handlers map[string]EventHandler
}

func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		clients: 	make(ClientList),
		handlers: 	make(map[string]EventHandler),
		otps:		NewRetentionMap(ctx, 5*time.Second),
	}
	
	m.setupEventHandlers()
	return m
}

func (m *Manager) setupEventHandlers() {
	m.handlers[EventSendMessage] = SendMessage
	m.handlers[EventChangeRoom] = ChatRoomHandler
}

func ChatRoomHandler(event Event, c *Client) error {
	var changeRoomEvent ChangeRoomEvent
	
	if err := json.Unmarshal(event.Payload, &changeRoomEvent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}
	
	c.chatroom = changeRoomEvent.Name
	
	return nil
}


func SendMessage( event Event, c *Client) error {
	var chatEvent SendMessageEvent 
	
	if err := json.Unmarshal(event.Payload, &chatEvent); err != nil {
		return fmt.Errorf("bad payload in request: %v", err)
	}
	
	var broadMessage NewMessageEvent
	
	broadMessage.Sent = time.Now()
	broadMessage.Message = chatEvent.Message
	broadMessage.From = chatEvent.From 
	
	data, err := json.Marshal(broadMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast message: %v", err)
	}
	
	outgoingEvent := Event{
		Payload: data,
		Type: EventNewMessage,
	}
	
	for client := range c.manager.clients {
		if client.chatroom == c.chatroom {
			client.egress <-outgoingEvent
		}
	}
	
	return nil
}

func (m *Manager) routeEvent(event Event, c *Client) error {
	// check if the event type exists and handle it
	if handler, ok := m.handlers[event.Type]; ok {
		if err := handler(event, c); err != nil {
			return err
		}
		return nil
	} else {
		return errors.New("there is no such event type")
	}
}

func (m *Manager) serveWS(w http.ResponseWriter, r *http.Request) {
	
	otp := r.URL.Query().Get("otp")
	if otp == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return 
	}
	
	if !m.otps.VerifyOTP(otp) {
		w.WriteHeader(http.StatusUnauthorized)
		return 
	}
	
	// if you get here then you have valid credentials and will be allowed to connect to websocket
	log.Println("new connection")
	
	// upgrade regular http connection into websocket
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return 
	}
	
	// create new client and pass in new connnection that was just created and pass in manager
	client := NewClient(conn, m)
	
	// add new client to manager ClietList
	m.addClient(client)
	
	// start client processes 
	go client.readMessages()
	go client.writeMessages()
}

func (m *Manager) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userLoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	// accept request 
	var req userLoginRequest 
	
	// decode request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return 
	}
	
	// check if request can be validated here
	// here is temp authentication replace with real authentication later
	if req.Username == "fernando" && req.Password == "123" { // hardcoded credentials for now
		type response struct {
			OTP string `json:"otp"` // use OTP it is expected from index.html func login() {}.then((data) => ...)
		}
		
		// from manager go to retention map and make newOTP
		otp := m.otps.NewOTP()
		
		resp := response{
			OTP : otp.Key,
		}
		
		data, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			return 
		}
		
		// return new OTP to user
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return 
	}
	
	// if we get here then user not authorized
	w.WriteHeader(http.StatusUnauthorized)
}

func (m *Manager) addClient(client *Client) {
	m.Lock() // Lock manager to prevent two people trying to connect at the same time avoid collision
	defer m.Unlock()
	
	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()
	
	if _, ok := m.clients[client]; ok {
		client.connection.Close()
		delete(m.clients, client)
	}
}


// check and limit where request come from and how they connect. 
// this is in order to prevent cross site request forgery
func checkOrigin(r *http.Request) bool { // this is the function signature that the websocket upgrader will expect
	origin := r.Header.Get("Origin")
	
	switch origin { // in real application you should make origin configurable from environment variable
	case "https://localhost:8080": // for now we will allow anything from port localhost:8080
		return true // accept connection
	default: 
		return false // ignore/ dismiss connection
	}
}