package main

import (
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"strings"

	"github.com/nicolagi/pine/ring"
	log "github.com/sirupsen/logrus"
)

func pipe(in net.Conn, out net.Conn) {
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	logger := log.WithFields(log.Fields{
		"op":  "pipe",
		"in":  in.RemoteAddr(),
		"out": out.LocalAddr(),
	})
	buffer, err := ring.NewMessageBuffer()
	if err != nil {
		logger.WithField("cause", err).Error("Could not create message buffer")
		return
	}
	logger.Info("Starting net pipe")
	chunk := make([]byte, 256)
	for {
		n, err := in.Read(chunk)
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			// Fragile. The error is poll.ErrNetClosing but it's in an internal package.
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			logger.WithField("cause", err).Error("Could not read")
			return
		}
		if err := buffer.Ingest(chunk[:n]); err != nil {
			logger.WithField("cause", err).Warning("Failed ingesting")
		} else if err := buffer.PrintMessages(os.Stdout); err != nil {
			logger.WithField("cause", err).Warning("Failed logging ingested messages")
		}
		for n > 0 {
			m, err := out.Write(chunk[:n])
			if err != nil {
				logger.WithField("cause", err).Error("Could not write")
				return
			}
			n -= m
		}
	}
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	var lnet, laddr, rnet, raddr string
	flag.StringVar(&lnet, "lnet", "tcp", "local listen address network `type`")
	flag.StringVar(&laddr, "l", "", "local listen `address`")
	flag.StringVar(&rnet, "rnet", "tcp", "remote connect address network `type`")
	flag.StringVar(&raddr, "r", "", "remote connect `address`")
	flag.Parse()

	logger := log.WithFields(log.Fields{
		"lnet":  lnet,
		"laddr": laddr,
		"rnet":  rnet,
		"raddr": raddr,
	})

	if laddr == "" || raddr == "" {
		flag.Usage()
		os.Exit(1)
	}

	listener, err := net.Listen(lnet, laddr)
	if err != nil && lnet == "unix" {
		listener, err = retryIfStaleUnixSocket(err, laddr)
	}
	if err != nil {
		logger.WithField("cause", err).Fatal("Could not listen")
	}
	for {
		local, err := listener.Accept()
		if err != nil {
			logger.WithField("cause", err).Error("Could not accept")
			continue
		}
		remote, err := net.Dial(rnet, raddr)
		if err != nil {
			_ = local.Close()
			logger.WithField("cause", err).Error("Could not connect")
			continue
		}
		go pipe(remote, local)
		go pipe(local, remote)
	}
}
