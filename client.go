package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"time"
)

type Request2 struct {
	Method string `json:"method"`
	Params string `json:"params"`
}

type Response2 struct {
	Result interface{} `json:"result"`
}

func main() {
	ipv4addr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250), Port: 5352}

	conn, err := net.ListenUDP("udp", ipv4addr)

	if err != nil {
		fmt.Println(err)
	}

	pc := ipv4.NewPacketConn(conn)

	// assume your have a interface named wlan
	iface, err := net.InterfaceByName("VMware Network Adapter VMnet8")
	if err != nil {
		fmt.Println(err)
	}
	if err := pc.JoinGroup(iface, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250)}); err != nil {
		fmt.Println(err)
	}

	if loop, err := pc.MulticastLoopback(); err == nil {
		fmt.Printf("MulticastLoopback status:%v\n", loop)
		if !loop {
			if err := pc.SetMulticastLoopback(true); err != nil {
				fmt.Printf("SetMulticastLoopback error:%v\n", err)
			}
		}
	}

	go func(conn *net.UDPConn) {
		for {
			buf := make([]byte, 4096)
			now := time.Now()
			err := conn.SetDeadline(now.Add(time.Second * 5))
			if err != nil {
				return
			}
			n, addr, err := conn.ReadFromUDP(buf)
			response := Response2{}
			//fmt.Println(buf[:n])
			err = json.Unmarshal(buf[:n], &response)

			if err != nil {
				fmt.Println(err)
			}

			if response.Result == nil {
				continue
			}

			fmt.Println("\ngo message from udp:", addr, response.Result.(map[string]interface{})["message"])
		}
	}(conn)

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Input: ")
		t, _, _ := reader.ReadLine()
		if len(t) == 0 {
			fmt.Println("Bye")
			os.Exit(0)
		}
		fmt.Println("Got input:", t)
		temp := string(t)
		var req Request2
		req.Method = "Say"
		req.Params = temp
		//fmt.Println(req)
		b, err := json.Marshal(&req)
		if err != nil {
			fmt.Println(err)
		}
		//fmt.Println("b is",b)
		n, err := conn.WriteToUDP(b, ipv4addr)
		fmt.Println("Sent to udp:", n, err)
	}

}
