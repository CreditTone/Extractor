package main

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"zhongguo/extractor"
)

func main() {
	rpc.Register(newExtractor())
	l, err := net.Listen("tcp", ":8585")
	if err != nil {
		fmt.Printf("Listener tcp err: %s", err)
		return
	}

	for {
		fmt.Println("wating...")
		conn, err := l.Accept()
		if err != nil {
			fmt.Sprintf("accept connection err: %s\n", conn)
		}
		go jsonrpc.ServeConn(conn)
	}
}

func newExtractor() *extractor.Extractor {
	extractor := extractor.NewExtractor()
	filter := func(config string) (string, bool) {
		return "", false
	}
	extractor.Filter = filter
	doTemplate := func(template, value string) string {
		return ""
	}
	extractor.DoTemplate = doTemplate
	return extractor
}
