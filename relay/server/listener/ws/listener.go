package ws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/netbirdio/netbird/relay/server/listener"
)

var (
	upgrader = websocket.Upgrader{} // use default options
)

type Listener struct {
	address string

	wg       sync.WaitGroup
	server   *http.Server
	acceptFn func(conn net.Conn)
}

func NewListener(address string) listener.Listener {
	return &Listener{
		address: address,
	}
}

// Listen todo: prevent multiple call
func (l *Listener) Listen(acceptFn func(conn net.Conn)) error {
	l.acceptFn = acceptFn
	http.HandleFunc("/", l.onAccept)

	l.server = &http.Server{
		Addr: l.address,
	}

	log.Debugf("WS server is listening on address: %s", l.address)
	err := l.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (l *Listener) Close() error {
	if l.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debugf("closing WS server")
	if err := l.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %v", err)
	}

	l.wg.Wait()
	return nil
}

func (l *Listener) onAccept(writer http.ResponseWriter, request *http.Request) {
	l.wg.Add(1)
	defer l.wg.Done()

	wsConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Errorf("failed to upgrade connection: %s", err)
		return
	}
	conn := NewConn(wsConn)
	l.acceptFn(conn)
	return
}
