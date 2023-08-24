

package main

import (
	"context"
	"log"
	"net/http"
)


func main() {
	setupAPI()
	
	// this one is used for http:
	// log.Fatal(http.ListenAndServe(":8080", nil))
	
	// to use https:
	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}

func setupAPI() {
	
	ctx := context.Background()
	
	manager := NewManager(ctx)
	
	
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
	http.HandleFunc("/login", manager.loginHandler)
}