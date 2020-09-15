package main

import (
	"bytes"
	"context"
	"github.com/Vilsol/tunnel-among-us/config"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net"
	"strconv"
	"time"
)

const maxBufferSize = 1024

var discoveryPacket []byte

func main() {
	config.InitializeConfig()

	discoveryPacket = append([]byte{0x04, 0x02}, []byte(viper.GetString("server.name")+"~Open~1~")...)

	level, err := log.ParseLevel(viper.GetString("log.level"))

	if err != nil {
		panic(err)
	}

	log.SetLevel(level)

	connect()
}

func connect() {
	address := "ws://" + viper.GetString("socket.host") + ":" + strconv.Itoa(viper.GetInt("socket.port"))
	conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), address)

	if err != nil {
		log.Error("Error dialing: ", err)
	} else {
		log.Info("Connected to: ", address)

		err, sender, receiver := server()

		go func() {
			for {
				raddr, err := net.ResolveUDPAddr("udp", viper.GetString("broadcast.host")+":"+strconv.Itoa(viper.GetInt("broadcast.port")))

				if err != nil {
					log.Error(errors.Wrap(err, "error resolving address"))
					break
				}

				conn, err := net.DialUDP("udp", nil, raddr)

				_, err = io.Copy(conn, bytes.NewReader(discoveryPacket))

				if err != nil {
					log.Error(errors.Wrap(err, "error sending discovery"))
					break
				}

				time.Sleep(time.Second)
			}
		}()

		go func() {
			for {
				msg, err := wsutil.ReadServerBinary(conn)

				log.Debugf("[%s] -> %s", conn.RemoteAddr().String(), msg)

				if err != nil {
					log.Error("Error reading message: ", err)
					close(sender)
					return
				}

				if len(msg) == 0 {
					continue
				}

				sender <- msg
			}
		}()

		for {
			msg, ok := <-receiver

			if !ok {
				break
			}

			log.Debugf("[%s] <- %s", conn.RemoteAddr().String(), msg)

			err = wsutil.WriteClientBinary(conn, msg)
			if err != nil {
				log.Error("Error writing message: ", err)
				return
			}
		}

		log.Infof("Closing connection to: %s", conn.RemoteAddr().String())
	}
}

func server() (error, chan []byte, chan []byte) {
	listener, err := net.ListenPacket("udp", "127.0.0.1:"+strconv.Itoa(viper.GetInt("server.port")))

	if err != nil {
		return errors.Wrap(err, "error listening to packets"), nil, nil
	}

	messageSender := make(chan []byte, 10)
	messageReceiver := make(chan []byte, 10)

	closer := make(chan bool, 2)

	clientAddrChannel := make(chan *net.Addr)

	go func() {
		defer listener.Close()
		defer close(closer)

		clientAddr := <-clientAddrChannel

		for {
			message, ok := <-messageSender

			if !ok {
				break
			}

			if _, err := listener.WriteTo(message, *clientAddr); err != nil {
				log.Error("Error sending packet: ", err)
				return
			}
		}
	}()

	go func() {
		defer listener.Close()
		defer close(messageReceiver)

		sentClientAddr := false

		for {
			select {
			case _ = <-closer:
				return
			default:
				break
			}

			buffer := make([]byte, maxBufferSize)
			length, clientAddr, err := listener.ReadFrom(buffer)

			if !sentClientAddr {
				sentClientAddr = true
				clientAddrChannel <- &clientAddr
			}

			if err != nil {
				log.Error("Error receiving packet: ", err)
				return
			}

			messageReceiver <- buffer[:length]
		}
	}()

	return nil, messageSender, messageReceiver
}
