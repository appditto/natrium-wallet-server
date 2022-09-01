package net

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	guuid "github.com/google/uuid"
	"github.com/recws-org/recws"
	"k8s.io/klog/v2"
)

type wsSubscribe struct {
	Action  string              `json:"action"`
	Topic   string              `json:"topic"`
	Ack     bool                `json:"ack"`
	Id      string              `json:"id"`
	Options map[string][]string `json:"options"`
}

type ConfirmationResponse struct {
	Topic   string                 `json:"topic"`
	Time    string                 `json:"time"`
	Message map[string]interface{} `json:"message"`
}

type WSCallbackBlock struct {
	Type           string `json:"type"`
	Account        string `json:"account"`
	Previous       string `json:"previous"`
	Representative string `json:"representative"`
	Balance        string `json:"balance"`
	Link           string `json:"link"`
	LinkAsAccount  string `json:"link_as_account"`
	Work           string `json:"work"`
	Signature      string `json:"signature"`
	Destination    string `json:"destination"`
	Source         string `json:"source"`
	Subtype        string `json:"subtype"`
}

type WSCallbackMsg struct {
	IsSend  string          `json:"is_send"`
	Block   WSCallbackBlock `json:"block"`
	Account string          `json:"account"`
	Hash    string          `json:"hash"`
	Amount  string          `json:"amount"`
}

func StartNanoWSClient(wsUrl string, callbackChan *chan *WSCallbackMsg) {
	ctx, cancel := context.WithCancel(context.Background())
	sentSubscribe := false
	ws := recws.RecConn{}
	// Nano subscription request
	subRequest := wsSubscribe{
		Action: "subscribe",
		Topic:  "confirmation",
		Ack:    false,
		Id:     guuid.New().String(),
		// ! TODO - subscribe to only connected acounts
		// Options: map[string][]string{
		// 	"accounts": {
		// 		account,
		// 	},
		// },
	}
	ws.Dial(wsUrl, nil)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer func() {
		signal.Stop(sigc)
		cancel()
	}()

	for {
		select {
		case <-sigc:
			cancel()
			return
		case <-ctx.Done():
			go ws.Close()
			klog.Infof("Websocket closed %s", ws.GetURL())
			return
		default:
			if !ws.IsConnected() {
				sentSubscribe = false
				klog.Infof("Websocket disconnected %s", ws.GetURL())
				time.Sleep(2 * time.Second)
				continue
			}

			// Sent subscribe with ack
			if !sentSubscribe {
				if err := ws.WriteJSON(subRequest); err != nil {
					klog.Infof("Error sending subscribe request %s", ws.GetURL())
					time.Sleep(2 * time.Second)
					continue
				} else {
					sentSubscribe = true
				}
			}

			var confMessage ConfirmationResponse
			err := ws.ReadJSON(&confMessage)
			if err != nil {
				klog.Infof("Error: ReadJSON %s", ws.GetURL())
				sentSubscribe = false
				continue
			}

			// Trigger callback
			if confMessage.Topic == "confirmation" {
				var deserialized WSCallbackMsg
				serialized, err := json.Marshal(confMessage.Message)
				if err != nil {
					klog.Infof("Error: Marshal ws %v", err)
					continue
				}
				if err := json.Unmarshal(serialized, &deserialized); err != nil {
					klog.Errorf("Error: decoding the callback to WSCallbackMsg %v", err)
					continue
				}
				*callbackChan <- &deserialized
			}
		}
	}
}
