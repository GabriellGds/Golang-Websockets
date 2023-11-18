package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nhooyr.io/websocket"
)

type Client struct {
	Nickname string
	conn     *websocket.Conn
	ctx      context.Context
}

type Message struct {
	From    string
	Content string
	SentAt  string
}

var (
	clients     map[*Client]bool = make(map[*Client]bool)
	joinCh      chan *Client     = make(chan *Client)
	broadcastCh chan Message     = make(chan Message)
)

func wsHandler(w http.ResponseWriter, r *http.Request) {
	nicname := r.URL.Query().Get("nickname")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	go writer()
	go joiner()

	client := Client{Nickname: nicname, conn: conn, ctx: r.Context()}
	joinCh <- &client

	reader(&client)
}

func writer() {
	for msg := range broadcastCh {
		for client := range clients {
			msg, err := json.Marshal(msg)
			if err != nil {
				log.Fatal(err)
			}
			client.conn.Write(client.ctx, websocket.MessageText, msg)
		}
	}
}

func reader(client *Client) {
	for {
		_, data, err := client.conn.Read(client.ctx)
		if err != nil {
			log.Println("closing client connection")
			delete(clients, client)
			msg := Message{From: client.Nickname, Content: client.Nickname + " saiu do chat", SentAt: time.Now().Format("02-01-2006 15:04:05")}

			broadcastCh <- msg
			break
		}

		var msgR Message
		json.Unmarshal(data, &msgR)

		broadcastCh <- Message{From: msgR.From, Content: msgR.Content, SentAt: time.Now().Format("02-01-2006 15:04:05")}
	}
}

func joiner() {
	for client := range joinCh {
		clients[client] = true

		broadcastCh <- Message{From: client.Nickname, Content: "O usuario " + client.Nickname + " se conectou", SentAt: time.Now().Format("02-01-2006 15:04:05")}
	}
}

func main() {

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		var res []*Client

		for c := range clients {
			res = append(res, c)
		}
		json.NewEncoder(w).Encode(res)
	})

	
	log.Fatal(http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil))
}
