package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

func tcp_handler(tcp_conn *net.TCPConn, cfg *Config) {
	defer tcp_conn.Close()

	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	ws_conn, _, err := dialer.Dial(cfg.RemoteAddr, nil)
	if err != nil {
		log.Printf("Fail to connect remote: err=%v", err)
		return
	}
	defer ws_conn.Close()

	reader := bufio.NewReader(tcp_conn)
	writer := bufio.NewWriter(tcp_conn)

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
					cfg.RemoteAddr,
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

Loop:
	for {
		select {
		case <-ch_stop:
			log.Printf("Connection stopped: remote=%s", cfg.RemoteAddr)
			break Loop
		case err := <-ch_err:
			select {
			case <-ch_stop:
				log.Printf("Connection stopped: remote=%s", cfg.RemoteAddr)
			default:
				log.Printf("Error occurred: remote=%s, err=%v", cfg.RemoteAddr, err)
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

func StartClient(cfg *Config) {
	log.Printf("Running in client mode: remote=%s", cfg.RemoteAddr)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port))
	if err != nil {
		log.Fatalf("Fail to listen on port %d: err=%v", cfg.Port, err)
	}
	log.Printf("Start listening on port %d", cfg.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Fail to accept connection: err=%v", err)
			continue
		}

		tcp_conn, ok := conn.(*net.TCPConn)
		if !ok {
			log.Printf("Fail to get TCP connection")
			conn.Close()
			continue
		}

		go tcp_handler(tcp_conn, cfg)
	}
}
