package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/ipv4"
	"net"
	"reflect"
)

type serviceMethod struct {
	method    reflect.Method
	argsType  reflect.Type
	replyType reflect.Type
}

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Response struct {
	Result interface{} `json:"result"`
}

var methods = make(map[string]*serviceMethod)

func addServer(server interface{}) error {
	serverType := reflect.TypeOf(server)
	for i := 0; i < serverType.NumMethod(); i++ {
		method := serverType.Method(i)

		methodType := method.Type

		args := methodType.In(1)

		reply := methodType.In(2)

		methods[method.Name] = &serviceMethod{
			method:    method,
			argsType:  args.Elem(),
			replyType: reply.Elem(),
		}
	}
	return nil
}

func getmethod(methodName string) (*serviceMethod, error) {
	method := methods[methodName]
	if method == nil {
		err := fmt.Errorf("rpc can't find the %v method", methodName)
		return nil, err
	}
	return method, nil
}

func startServer(server interface{}) error {
	ipv4addr := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250), Port: 5352}

	conn, err := net.ListenUDP("udp", ipv4addr)

	if err != nil {
		fmt.Println(err)
	}

	pc := ipv4.NewPacketConn(conn)

	// assume your have a interface named wlan
	iface, err := net.InterfaceByName("ens33")
	if err != nil {
		return err
	}
	if err := pc.JoinGroup(iface, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 250)}); err != nil {
		return err
	}

	if loop, err := pc.MulticastLoopback(); err == nil {
		fmt.Printf("MulticastLoopback status:%v\n", loop)
		if !loop {
			if err := pc.SetMulticastLoopback(true); err != nil {
				fmt.Printf("SetMulticastLoopback error:%v\n", err)
			}
		}
	}

	//go func(conn *net.UDPConn, ipv4addr *net.UDPAddr) {
	//	reader := bufio.NewReader(os.Stdin)
	//	for {
	//		fmt.Print("Input: ")
	//		t, _, _ := reader.ReadLine()
	//		if len(t) == 0 {
	//			fmt.Println("Bye")
	//			os.Exit(0)
	//		}
	//		fmt.Println("Got input:", t)
	//		temp := string(t)
	//		var req Request2
	//		req.Method = "Say"
	//		req.Params = temp
	//		fmt.Println(req)
	//		b, err := json.Marshal(&req)
	//		if err != nil{
	//			fmt.Println(err)
	//		}
	//		fmt.Println("b is",b)
	//		n, err := conn.WriteToUDP(b, ipv4addr)
	//		fmt.Println("Sent to udp:", n, err)
	//	}
	//}(conn, ipv4addr)

	for {
		var req Request
		var response Response
		buff := make([]byte, 4096)

		n, addr, err := conn.ReadFromUDP(buff)

		if err != nil {
			fmt.Println(err)
			continue
		}

		fmt.Printf("got message form %v:%v\n", addr.IP, addr.Port)

		// 将请求转化为json
		if err := json.Unmarshal(buff[:n], &req); err != nil {
			fmt.Printf("unmarshal err : %v\n", err)
			continue
		}

		//fmt.Println("req is",req)

		serverMethod, err := getmethod(req.Method)

		if err != nil {
			fmt.Println(err)
			continue
		}

		args := reflect.New(serverMethod.argsType)

		if err := json.Unmarshal(req.Params, args.Interface()); err != nil {
			fmt.Println(err)
		}
		fmt.Println("Successfully read params", args.String())

		reply := reflect.New(serverMethod.replyType)
		fmt.Println("Successfully read replytype", reply.Interface())

		errValue := serverMethod.method.Func.Call([]reflect.Value{
			reflect.ValueOf(server),
			args,
			reply,
		})

		fmt.Println("reply is", reply)

		var errResult error
		errInter := errValue[0].Interface()
		if errInter != nil {
			errResult = errInter.(error)
		}

		if errResult == nil {
			response.Result = reply.Interface()
		}

		fmt.Println("response is", response.Result)
		marshalByte, err := json.Marshal(&response)
		fmt.Println(marshalByte)
		if err != nil {
			fmt.Println(err)
		}

		//发送响应
		if _, err := conn.WriteToUDP(marshalByte, addr); err != nil {
			fmt.Println(err)
		}
	}
}

type Api struct {
}

type Result struct {
	Message string `json:"message"`
}

func (t *Api) Say(r *string, w *Result) error {
	*w = Result{
		Message: "Hello," + *r,
	}
	return nil
}

func main() {
	err := addServer(new(Api))
	if err != nil {
		fmt.Println(err)
	}
	err = startServer(new(Api))
	if err != nil {
		return
	}

}
