// +build linux netbsd

package main

import (
	"net"
	"os"
	"syscall"
)

func retryIfStaleUnixSocket(listenErr error, pathname string) (net.Listener, error) {
	if !isAddressAlreadyInUseError(listenErr) {
		return nil, listenErr
	}
	if _, err := net.Dial("unix", pathname); isConnectionRefusedError(err) {
		_ = os.Remove(pathname)
	}
	return net.Listen("unix", pathname)
}

func isAddressAlreadyInUseError(err error) bool {
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			return err.Err == syscall.EADDRINUSE
		}
	}
	return false
}

func isConnectionRefusedError(err error) bool {
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			return err.Err == syscall.ECONNREFUSED
		}
	}
	return false
}
