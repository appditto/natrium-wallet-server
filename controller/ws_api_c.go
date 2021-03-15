package controller

import (
	"github.com/gofiber/websocket/v2"
	"github.com/golang/glog"
)

func HandleWSMessage(c *websocket.Conn) {
	var (
		mt  int
		msg []byte
		err error
	)
	for {
		if mt, msg, err = c.ReadMessage(); err != nil {
			glog.Error("read: %s", err)
			break
		}
		glog.Infof("recv: %s", msg)

		if err = c.WriteMessage(mt, msg); err != nil {
			glog.Errorf("write: %s", err)
			break
		}
	}
}
