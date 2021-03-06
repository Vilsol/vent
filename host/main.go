package host

import (
	"bytes"
	"fmt"
	"github.com/Vilsol/vent/utils"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const maxBufferSize = 1024

var logWriter = make(chan string, 100)

func RunHost() {
	log.Info("Started vent host")

	if log.IsLevelEnabled(log.DebugLevel) {
		go func() {
			f, err := os.Create("packet.log")

			if err != nil {
				panic(err)
			}

			defer f.Close()

			for {
				line := <-logWriter

				if !strings.HasSuffix(line, "\n") {
					line = line + "\n"
				}

				f.WriteString(line)
				f.Sync()
			}
		}()
	}

	host()
}

func host() {
	log.Error(http.ListenAndServe(":"+strconv.Itoa(viper.GetInt("socket.port")), http.HandlerFunc(handleConnection)))
}

func handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Error("Error upgrading request: ", err)
		return
	}

	go func() {
		defer conn.Close()

		log.Infof("New connection from: %s", conn.RemoteAddr().String())

		err, sender, receiver := client()
		if err != nil {
			log.Error("Error creating socket: ", err)
			return
		}

		go func() {
			for {
				msg, err := wsutil.ReadClientBinary(conn)

				log.Debugf("[%s] -> %s", conn.RemoteAddr().String(), utils.BytesToHex(msg))
				if log.IsLevelEnabled(log.DebugLevel) {
					logWriter <- fmt.Sprintf("[%s] -> %s", conn.RemoteAddr().String(), utils.BytesToHex(msg))
				}

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

			log.Debugf("[%s] <- %s", conn.RemoteAddr().String(), utils.BytesToHex(msg))
			if log.IsLevelEnabled(log.DebugLevel) {
				logWriter <- fmt.Sprintf("[%s] <- %s", conn.RemoteAddr().String(), utils.BytesToHex(msg))
			}

			err = wsutil.WriteServerBinary(conn, msg)
			if err != nil {
				log.Error("Error writing message: ", err)
				return
			}
		}

		log.Infof("Closing connection to: %s", conn.RemoteAddr().String())
	}()
}

func client() (error, chan []byte, chan []byte) {
	raddr, err := net.ResolveUDPAddr("udp", viper.GetString("server.host")+":"+strconv.Itoa(viper.GetInt("server.port")))
	if err != nil {
		return errors.Wrap(err, "error resolving address"), nil, nil
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return errors.Wrap(err, "error dialing address"), nil, nil
	}

	messageSender := make(chan []byte, 10)
	messageReceiver := make(chan []byte, 10)

	closer := make(chan bool, 2)

	go func() {
		defer conn.Close()
		defer close(closer)

		for {
			message, ok := <-messageSender

			if !ok {
				break
			}

			if _, err := io.Copy(conn, bytes.NewReader(message)); err != nil {
				log.Error("Error sending packet: ", err)
				return
			}
		}
	}()

	go func() {
		defer conn.Close()
		defer close(messageReceiver)

		for {
			select {
			case _ = <-closer:
				return
			default:
				break
			}

			buffer := make([]byte, maxBufferSize)
			var length int
			if length, _, err = conn.ReadFromUDP(buffer); err != nil {
				log.Error("Error receiving packet: ", err)
				return
			}

			messageReceiver <- buffer[:length]
		}
	}()

	return nil, messageSender, messageReceiver
}
