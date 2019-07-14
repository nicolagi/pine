package main

import (
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"strings"

	"github.com/google/gops/agent"
	"github.com/nicolagi/pine/ring"
	log "github.com/sirupsen/logrus"
)

var (
	errInvalidAddr = errors.New("invalid addr, expected, e.g., tcp!localhost!4321 or unix!/path/file.sock")
)

func pipe(in net.Conn, out net.Conn, msize uint32) {
	defer func() {
		_ = in.Close()
		_ = out.Close()
	}()
	logger := log.WithFields(log.Fields{
		"op":  "pipe",
		"in":  in.RemoteAddr(),
		"out": out.LocalAddr(),
	})
	buffer, err := ring.NewMessageBuffer(msize)
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
			logger.WithField("cause", err).Warning("Failed ingesting - is the buffer size large enough? Check -msize option against [TR]version messages")
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

func splitAddr(s string) (network string, addr string, err error) {
	parts := strings.SplitN(s, "!", 3)
	if len(parts) < 2 {
		err = errInvalidAddr
		return
	}
	network = parts[0]
	switch network {
	case "tcp":
		if len(parts) != 3 {
			err = errInvalidAddr
			return
		}
		addr = parts[1] + ":" + parts[2]
	case "unix":
		if len(parts) != 2 {
			err = errInvalidAddr
			return
		}
		addr = parts[1]
	}
	return
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})

	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true,
	}); err != nil {
		log.WithField("cause", err).Warning("Could not start gops agent")
	}

	local := flag.String("local", "", "local address (TCP host and port or path to Unix socket)")
	remote := flag.String("remote", "", "remote address (TCP host and port or path to Unix socket)")
	msize := flag.Uint("msize", 8192, "9P message size")
	flag.Parse()

	logger := log.WithFields(log.Fields{
		"local":  *local,
		"remote": *remote,
		"msize":  *msize,
	})

	if *local == "" || *remote == "" || *msize == 0 {
		flag.Usage()
		os.Exit(1)
	}

	lnet, laddr, err := splitAddr(*local)
	if err != nil {
		logger.WithField("cause", err).Fatal("Could not understand local addr")
	}
	rnet, raddr, err := splitAddr(*remote)
	if err != nil {
		logger.WithField("cause", err).Fatal("Could not understand remote addr")
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
		go pipe(remote, local, uint32(*msize))
		go pipe(local, remote, uint32(*msize))
	}
}
