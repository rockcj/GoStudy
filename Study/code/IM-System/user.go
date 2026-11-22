package main

import "net"

// 用户类
type User struct {
	Name string
	Addr string      //客户端地址
	C    chan string //用于通信的管道(双向通道，可读可写)  (<-chan只读 cahn<-只写)
	conn net.Conn    //客户端的连接
}

// 创建一个用户的API
func NewUser(conn net.Conn) *User {
	//获取客户端地址
	userAddr := conn.RemoteAddr().String()

	//初始化对象/构造方法：
	user := &User{
		Name: userAddr, //先默认名字为地址
		Addr: userAddr,
		C:    make(chan string), //创建管道
		conn: conn,              //把当前的连接存入对象即可
	}

	//创建完之后立马启动监听当前user channel的goroutine
	go user.ListenMessage()

	return user
}

// 创建一个监听当前对象channel的goroutine方法 用于监听服务端消息并且实时发送给客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C                     //读取C(管道中)的数据
		this.conn.Write([]byte(msg + "\n")) //将msg写入管道/相当于发送数据
	}
}
