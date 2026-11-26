package main

func main() {
	//调用server.go中写好的Server服务
	server := NewServer("127.0.0.1", 8888)
	server.Start()
}
