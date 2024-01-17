package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		EnableCompression: true,
		ReadBufferSize:    10240,
		WriteBufferSize:   10240,
	}
)

func ws_handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("New connection: remote=%s", r.RemoteAddr)

	ws_conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Fail to handshake: remote=%s, err=%v", r.RemoteAddr, err)
		return
	}
	defer ws_conn.Close()

	rdp_conn, err := net.Dial("tcp", "127.0.0.1:3389")
	if err != nil {
		log.Printf("Fail to connect RDP: remote=%s, err=%v", r.RemoteAddr, err)
		return
	}
	defer rdp_conn.Close()

	reader := bufio.NewReader(rdp_conn)
	writer := bufio.NewWriter(rdp_conn)

	ch_ws := make(chan []byte)
	ch_rdp := make(chan []byte)
	ch_err := make(chan error)
	ch_stop := make(chan struct{}, 1)

	go func() {
		ws_conn.SetCloseHandler(func(code int, text string) error {
			ch_stop <- struct{}{}

			message := websocket.FormatCloseMessage(code, "")
			log.Println(
				fmt.Sprintf(
					"Disconnected: remote=%s, code=%d, msg=%s, text=%s",
					r.RemoteAddr,
					code,
					message,
					text),
			)

			err := ws_conn.WriteControl(
				websocket.CloseMessage,
				message,
				time.Now().Add(time.Second),
			)
			if err != nil && err != websocket.ErrCloseSent {
				return err
			}

			return nil
		})

		for {
			_, msg, err := ws_conn.ReadMessage()
			if err != nil {
				ch_err <- err
				break
			}

			ch_ws <- msg
		}
	}()

	go func() {
		for {
			buf := make([]byte, 10240)
			n, err := reader.Read(buf)
			if err != nil {
				if err == net.ErrClosed {
					ch_stop <- struct{}{}
				} else {
					ch_err <- err
				}

				break
			}

			ch_rdp <- buf[:n]
		}
	}()

	log.Printf("Tunnel constructed: remote=%s", r.RemoteAddr)

Loop:
	for {
		select {
		case <-ch_stop:
			log.Printf("Connection stopped: remote=%s", r.RemoteAddr)
			break Loop
		case err := <-ch_err:
			select {
			case <-ch_stop:
				log.Printf("Connection stopped: remote=%s", r.RemoteAddr)
			default:
				log.Printf("Error occurred: remote=%s, err=%v", r.RemoteAddr, err)
			}
			break Loop
		case msg := <-ch_ws:
			_, err := writer.Write(msg)
			if err != nil {
				ch_err <- err
				break
			}

			err = writer.Flush()
			if err != nil {
				ch_err <- err
				break
			}
		case msg := <-ch_rdp:
			err := ws_conn.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				ch_err <- err
				break
			}
		}
	}
}

func StartServer(cfg *Config) {
	log.Println("Running in server mode")

	pair, err := GenerateKeyPair()
	if err != nil {
		log.Fatalf("Fail to generate key pair: err=%v", err)
	}

	if err := WriteKeyPair(pair, "httptun"); err != nil {
		log.Fatalf("Fail to write key pair files: err=%v", err)
	}

	http.HandleFunc("/", ws_handler)

	log.Printf("Start listening on port %d", cfg.Port)
	log.Fatalln(
		http.ListenAndServeTLS(
			fmt.Sprintf("0.0.0.0:%d", cfg.Port),
			"httptun.cert",
			"httptun.key",
			nil),
	)
}
