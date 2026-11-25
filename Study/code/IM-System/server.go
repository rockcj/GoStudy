package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

type Server struct {
	Ip   string
	Port int

	//创建在线用户列表
	OnlineMap map[string]*User //键值对

	//记得加上读写锁 （防止同时进行读写操作）（防止读写混乱）
	mapLock sync.RWMutex //一般有关同步的机制都在sync里，有很多相关的锁

	//消息广播的channel
	Message chan string
}

// 创建server的接口  (普通接口)
func NewServer(ip string, port int) *Server {
	server := &Server{ //通过Server结构体(相当于类)new一个server对象，并且返回其对象地址
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string), //广播消息，要将其发送给每个用户
	}
	return server //返回该对象（指针类型）
}

// 监听Message广播消息channel的goroutine,一旦有消息就发送给全部的在线用户
func (this *Server) ListenMessage() {
	for {
		msg := <-this.Message //尝试不断地从Message管道中读取数据
		//将msg发送给全部的在线User
		this.mapLock.Lock()                  //先加锁
		for _, cli := range this.OnlineMap { //因为我们不需要也不处理key,所以使用 ‘_,’忽略即可，我们处理value即可
			cli.C <- msg //将msg写入用户的C管道即可 这样就将消息发出去了
		}
		this.mapLock.Unlock() //解锁
	}
}

// 消息广播方法  传入形参：消息是由哪个用户发起的，消息内容是什么
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	this.Message <- sendMsg //将sendmsg写入到广播消息的管道中

	//此时还应该有一个go程不断地从Message中读数据将其发送给所有用户 (写在start方法中，一开机就启动监听)
}

// 业务处理方法
func (this *Server) Handler(conn net.Conn) {
	//当前连接的业务
	fmt.Println("连接构建成功")
	//首先创建一个user对象,传入当前server的this指针
	user := NewUser(conn, this)

	//1.用户上线
	user.Online()

	//3.接收客户端发送的消息
	go func() { //go一个匿名方法接收信息
		buf := make([]byte, 4096) //初始化接受字符串的长度
		for {
			n, err := conn.Read(buf) //返回的n为成功读取的长度
			if n == 0 {
				user.Offline() //用户下线
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("连接读取错误：", err)
				return
			}

			//提取用户的消息（去除'\n'）
			//改方法同时兼容windows和Linux
			msg := strings.TrimSpace(string(buf[:n])) //去掉行尾的 \r\n，Windows 的 nc 回车发的是 CRLF  Linux的nc发送的是LF格式

			//将得到的消息进行广播
			user.DoMessage(msg) //用户进行处理
		}
	}()
	//当前handler阻塞即可
	select {}
}

// 启动服务器的接口  （当前类的一个成员方法）
func (this *Server) Start() {
	//socket listen操作
	//返回一个listener对象，初始化并赋值
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port)) //该方法传入服务器的网络类型tcp/udp/tcp4/tcp6等等，和服务器地址url（ip+端口）
	//异常处理
	if err != nil {
		//打印异常
		fmt.Println("net.Listen err:", err)
		return
	}
	//close listen socket操作
	defer listener.Close() //压栈，当整个函数执行完则关闭监听窗口，防止忘记

	//启动监听Message的goroutine
	go this.ListenMessage()

	//直接来一个大（死）循环来完成实时监听，因为一个服务器应该不需要来阻塞该监听循环
	for {
		//accept,接受请求则返回一个连接对象
		conn, err := listener.Accept()
		//处理异常
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue //为了要进行进行正常服务器连接通讯，选用continue,不要退出服务
		}

		//do handler 业务回调，正常拿到客户端的请求消息后，做业务处理
		go this.Handler(conn) //为了不阻塞服务器通讯，应该go一个 “协程” 出来运行handler业务
	}
}

//方法名大写的位公开方法，小写的为私有方法
//同一个main包下，只要有一个main方法就够了，不需要/不能够/不必要 每个main包下的文件都出现main方法
