package client

import (
	"context"
	"github.com/Vilsol/vent/utils"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"net"
	"strconv"
	"time"
)

type Message struct {
	Address *net.Addr
	Payload []byte
}

const maxBufferSize = 1024

var discoveryPacket []byte

var messageSender chan *Message

func RunClient() {
	log.Info("Started vent client")

	discoveryPacket = append([]byte{0x04, 0x02}, []byte(viper.GetString("server.name")+"~Open~1~")...)

	connect()
}

func connect() {
	go broadcaster()

	err := server()

	if err != nil {
		panic(err)
	}
}

func server() error {
	listener, err := net.ListenPacket("udp", ":"+strconv.Itoa(viper.GetInt("server.port")))

	if err != nil {
		return errors.Wrap(err, "error listening to packets")
	}

	messageSender = make(chan *Message, 50)
	closer := make(chan bool, 2)

	go func() {
		defer close(closer)

		for {
			message, ok := <-messageSender

			if !ok {
				break
			}

			if _, err := listener.WriteTo(message.Payload, *message.Address); err != nil {
				log.Error("Error sending packet: ", err)
				return
			}
		}
	}()

	for {
		buffer := make([]byte, maxBufferSize)
		length, clientAddr, err := listener.ReadFrom(buffer)

		if err != nil {
			return errors.Wrap(err, "error receiving packet: ")
		}

		client := getClient(&clientAddr)
		client.Queue <- buffer[:length]
	}
}

type Client struct {
	Connection *net.Conn
	Queue      chan []byte
}

var clients = make(map[string]*Client)

func getClient(address *net.Addr) *Client {
	if client, ok := clients[(*address).String()]; ok {
		return client
	}

	sockAddress := "ws://" + viper.GetString("socket.host") + ":" + strconv.Itoa(viper.GetInt("socket.port"))
	conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), sockAddress)

	if err != nil {
		panic(err)
	}

	log.Infof("New client: %s", *address)

	client := &Client{
		Connection: &conn,
		Queue:      make(chan []byte, 10),
	}

	clients[(*address).String()] = client

	go func() {
		for {
			msg, err := wsutil.ReadServerBinary(conn)

			log.Debugf("[%s] -> %s", conn.RemoteAddr().String(), utils.BytesToHex(msg))

			if err != nil {
				log.Error("Error reading message: ", err)
				return
			}

			if len(msg) == 0 {
				continue
			}

			messageSender <- &Message{
				Address: address,
				Payload: msg,
			}
		}
	}()

	go func() {
		for {
			msg, ok := <-client.Queue

			if !ok {
				break
			}

			log.Debugf("[%s] <- %s", conn.RemoteAddr().String(), utils.BytesToHex(msg))

			err = wsutil.WriteClientBinary(conn, msg)
			if err != nil {
				log.Error("Error writing message: ", err)
				return
			}
		}
	}()

	return client
}

func broadcaster() {
	ifaces, err := net.Interfaces()

	if err != nil {
		panic(err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if n, ok := addr.(*net.IPNet); ok {
					if n.IP.To4() != nil {
						log.Infof("Broadcasting game on interface: %s - %s", iface.Name, addr.String())
						break
					}
				}
			}
		}
	}

	pc, err := net.ListenPacket("udp4", ":32420")

	if err != nil {
		panic(err)
	}

	defer pc.Close()

	validAddresses := make(map[string]bool)

	for _, i := range ifaces {
		addresses, err := i.Addrs()
		if err == nil {
			for _, addr := range addresses {
				if n, ok := addr.(*net.IPNet); ok {
					if n.IP.To4() != nil {
						broadcastIp := net.ParseIP(n.IP.String())
						broadcastIp[15] = 255
						validAddresses[broadcastIp.String()] = true
					}
				}
			}
		}
	}

	for {
		for broadcastIp := range validAddresses {
			log.Debugf("Broadcasting to: %s", broadcastIp)

			addr, err := net.ResolveUDPAddr("udp4", broadcastIp+":"+strconv.Itoa(viper.GetInt("broadcast.port")))
			if err != nil {
				log.Warnf("Error broadcasting to %s. Excluded from further broadcasts: %s", broadcastIp, err)
				delete(validAddresses, broadcastIp)
				continue
			}

			_, err = pc.WriteTo(discoveryPacket, addr)
			if err != nil {
				log.Warnf("Error broadcasting to %s. Excluded from further broadcasts: %s", broadcastIp, err)
				delete(validAddresses, broadcastIp)
				continue
			}
		}

		time.Sleep(time.Second)
	}
}
