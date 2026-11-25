package main

import (
	"net"
)

// 用户类
type User struct {
	Name string
	Addr string      //客户端地址
	C    chan string //用于通信的管道(双向通道，可读可写)  (<-chan只读 cahn<-只写)
	conn net.Conn    //客户端的连接

	server *Server //将用户关联上server，因为要知道当前用户是属于哪个server
}

// 创建一个用户的API，还要传入其是哪个server的
func NewUser(conn net.Conn, server *Server) *User {
	//获取客户端地址
	userAddr := conn.RemoteAddr().String()

	//初始化对象/构造方法：
	user := &User{
		Name:   userAddr, //先默认名字为地址
		Addr:   userAddr,
		C:      make(chan string), //创建管道
		conn:   conn,              //把当前的连接存入对象即可
		server: server,
	}

	//创建完之后立马启动监听当前user channel的goroutine
	go user.ListenMessage()

	return user
}

// 用户上线
func (this *User) Online() {
	//1.用户上线，将用户添加到OnlineMap中
	this.server.mapLock.Lock()              //先上锁
	this.server.OnlineMap[this.Name] = this //键：user.Name   值：user对象
	this.server.mapLock.Unlock()            //记得关锁
	//2.广播用户上线的消息
	this.server.BroadCast(this, "已上线")
}

// 用户下线
func (this *User) Offline() {
	//1.用户下线，将用从OnlineMap中删除
	this.server.mapLock.Lock()               //先上锁
	delete(this.server.OnlineMap, this.Name) //删除OlineMap中的该name对应的键值对
	this.server.mapLock.Unlock()             //记得关锁
	//2.广播用户上线的消息
	this.server.BroadCast(this, "已下线")
}

// 用户发消息
func (u *User) SendMsg(msg string) {
	u.conn.Write([]byte(msg))
}

// 用户处理消息
func (this *User) DoMessage(msg string) {
	//加一个查询功能：
	if msg == "who" {
		//查询当前用户都有哪些
		this.server.mapLock.Lock()
		for _, user := range this.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]: " + user.Name + "在线...\n"
			//然后将消息发送给this(该)用户
			//this.C <- onlineMsg //方法1
			this.SendMsg(onlineMsg) //方法2
		}
		this.server.mapLock.Unlock()
	} else {
		//通过user调用其关联的server中的BroadCast即可
		this.server.BroadCast(this, msg)
	}
}

// 创建一个监听当前对象channel的goroutine方法 用于监听服务端消息并且实时发送给客户端
func (this *User) ListenMessage() {
	for {
		msg := <-this.C                     //读取C(管道中)的数据
		this.conn.Write([]byte(msg + "\n")) //将msg写入连接的管道/相当于发送数据
	}
}
