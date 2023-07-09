package server

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type wsDevice struct {
	mtx  sync.Mutex
	conn net.Conn
}

type wsDevices struct {
	mtx  sync.RWMutex
	data map[string]*wsDevice
}

var Devices = wsDevices{data: make(map[string]*wsDevice)}

func StartNewHTTPServer(httpPort int) error {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowMethods: []string{http.MethodPost, http.MethodGet},
	}))
	setupRoutes(e)

	ch := make(chan error)
	go func() {
		ch <- e.Start(fmt.Sprintf(":%d", httpPort))
	}()

	var err error
	select {
	case err = <-ch:
		return err
	case <-time.After(time.Second * 1):
	}
	log.Printf("HTTP server listening at %v", httpPort)
	return nil
}

func setupRoutes(e *echo.Echo) {
	e.GET("/ws", wsHandler)
	e.POST("/message", messageHandler)
}

// endpoint для получения сообщений
func messageHandler(c echo.Context) error {
	var message Message
	err := c.Bind(&message)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid input json")
	}
	if message.DeviceId != nil {
		Devices.mtx.RLock()
		defer Devices.mtx.RUnlock()
		dev, ok := Devices.data[*message.DeviceId]
		if !ok {
			return c.String(http.StatusNotFound, "device_id not registered")
		}
		msg, err := json.Marshal(message)
		if err != nil {
			return c.String(http.StatusBadRequest, "couldn't marshal message")
		}
		dev.mtx.Lock()
		defer dev.mtx.Unlock()
		err = wsutil.WriteServerMessage(dev.conn, 1, msg)
		if err != nil {
			return c.String(http.StatusServiceUnavailable, "message not delivered")
		}
		return c.NoContent(http.StatusOK)

	} else {
		msg, err := json.Marshal(message)
		if err != nil {
			return c.String(http.StatusBadRequest, "couldn't marshal message")
		}
		Devices.mtx.RLock()
		defer Devices.mtx.RUnlock()
		for _, dev := range Devices.data {
			go func(d *wsDevice) {
				d.mtx.Lock()
				defer d.mtx.Unlock()
				err = wsutil.WriteServerMessage(d.conn, 1, msg)
				if err != nil {
				}
			}(dev)
		}
		// при массовой рассылке сразу возвращаем 200, при необходимости можно прописать любую логику формирования ответа
		return c.NoContent(http.StatusOK)
	}
}

// endpoint для регистрации ws клиентов
func wsHandler(c echo.Context) error {
	// предположим, что устройство указывает свой device_id как query parameter
	device_id := c.QueryParam("device_id")
	if len(device_id) == 0 {
		return c.String(http.StatusUnauthorized, "device_id parameter required")
	}

	var conn net.Conn
	var err error
	// отклоняем повторное подключение с уже зарегистрированным device_id
	Devices.mtx.Lock()
	dev, registered := Devices.data[device_id]
	if registered {
		// проверим, активно ли еще зарегистрированное ранее соединение
		dev.mtx.Lock()
		err = wsutil.WriteServerMessage(dev.conn, 1, []byte("liveness probe"))
		if err != nil {
			_ = dev.conn.Close()
			registered = false
		}
		dev.mtx.Unlock()
	}

	if registered {
		Devices.mtx.Unlock()
		return c.String(http.StatusUnauthorized, "device_id already registered")
	} else {
		// переходим на ws и регистрируем device_id подключившегося устройства
		conn, _, _, err = ws.UpgradeHTTP(c.Request(), c.Response())
		if err != nil {
			Devices.mtx.Unlock()
			return c.String(http.StatusBadRequest, err.Error())
		}
		defer conn.Close()
		Devices.data[device_id] = &wsDevice{conn: conn}
	}
	Devices.mtx.Unlock()

	for {
		// просто читаем входящие сообщения
		msg, op, err := wsutil.ReadClientData(conn)
		if err != nil {
			log.Println(err.Error())
		}
		if op != 1 {
			// при выходе удаляем устройство из списка зарегистрированных
			Devices.mtx.Lock()
			defer Devices.mtx.Unlock()
			_ = conn.Close()
			delete(Devices.data, device_id)
			break
		}
		log.Println(msg)
	}
	return nil
}

type Message struct {
	DeviceId *string `json:"device_id,omitempty"`
	Id       *string `json:"id,omitempty"`
	Kind     *int    `json:"kind,omitempty"`
	Message  *string `json:"message,omitempty"`
}
