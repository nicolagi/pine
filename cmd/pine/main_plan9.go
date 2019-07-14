// +build plan9

package main

import "net"

func retryIfStaleUnixSocket(listenErr error, _ string) (net.Listener, error) {
	return nil, listenErr
}
