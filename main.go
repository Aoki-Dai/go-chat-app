package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// クライアントの接続を保持するマップ
var clients = make(map[*websocket.Conn]bool)

// ブロードキャストするメッセージを受け取るチャネル
var broadcast = make(chan Message)

// WebSocketコネクションのアップグレーダー
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // オリジン制限を無視（本番環境では注意が必要）
	},
}

// メッセージの構造体
type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func main() {
	// ルートハンドラー
	http.HandleFunc("/", handleHome)
	// WebSocket接続ハンドラー
	http.HandleFunc("/ws", handleConnections)

	// メッセージ処理を別のゴルーチンで実行
	go handleMessages()

	// サーバーの起動
	fmt.Println("Server is running on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// HTTP接続をWebSocket接続にアップグレード
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	// 新しいクライアントを登録
	clients[ws] = true

	for {
		var msg Message
		// メッセージの読み取り
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// 受信したメッセージをブロードキャストチャネルに送信
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// ブロードキャストチャネルからメッセージを受信
		msg := <-broadcast
		// 全てのクライアントにメッセージを送信
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
