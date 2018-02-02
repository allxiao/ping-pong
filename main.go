package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	flags "github.com/jessevdk/go-flags"
)

var opts struct {
	Mode    string `short:"m" long:"mode" description:"Run mode" default:"server" choice:"client" choice:"server"`
	Address string `short:"a" long:"address" description:"Listen address for the server, or connection endpoint for client" default:"localhost:3210"`
}

func startClient() {
	var frequency int64 = 1000

	var interval = time.Duration(int64(time.Second) / frequency)
	var closeChan = make(chan struct{})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			close(closeChan)
			break
		}
	}()
	count := 0

	defer func() {
		fmt.Println()
		log.Println("Established ", count)
	}()

	for {
		conn, err := net.Dial("tcp", opts.Address)
		if err != nil {
			log.Fatalln("Error dialing: ", err)
		}
		count++
		fmt.Print(".")

		if count%1000 == 0 {
			fmt.Println()
			log.Println("Established: ", count)
		}

		go func(conn net.Conn) {
			defer conn.Close()

			ticker := time.NewTicker(time.Second * 5)
			for {
				select {
				case <-ticker.C:
					conn.Write([]byte("ping\n"))
				case _, _ = <-closeChan:
					conn.Write([]byte("close\n"))
					return
				}
			}
		}(conn)

		select {
		case _, _ = <-closeChan:
			return
		default:
			time.Sleep(interval)
		}
	}
}

func startServer() {
	address := opts.Address
	if strings.HasPrefix(address, ":") {
		address = "0.0.0.0" + address
	}

	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalln("Error listening: ", err)
	}
	defer l.Close()
	log.Println("Listening on " + address)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln("Error accepting: ", err)
		}

		go func(conn net.Conn) {
			defer conn.Close()

			buf := make([]byte, 1024)

			for {
				reqLen, err := conn.Read(buf)
				if err != nil {
					log.Println("Error reading: ", err)
				}
				if reqLen == 0 {
					break
				}

				received := string(buf[:reqLen])
				switch received {
				case "ping\n":
					conn.Write([]byte("pong\n"))
				case "close\n":
					return
				default:
					log.Println("Unknown command: ", received)
				}
			}
		}(conn)
	}
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(-1)
	}

	if opts.Mode == "client" {
		startClient()
	} else {
		startServer()
	}
}