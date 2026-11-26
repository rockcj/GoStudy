package main

import (
	"net"
	"strings"
	"sync"
)

// 用户类
type User struct {
	Name string
	Addr string      //客户端地址
	C    chan string //用于通信的管道(双向通道，可读可写)  (<-chan只读 cahn<-只写)
	conn net.Conn    //客户端的连接

	server *Server //将用户关联上server，因为要知道当前用户是属于哪个server

	done      chan struct{}
	closeOnce sync.Once
}

// 用于销毁用户资源
func (u *User) Close() {
	u.closeOnce.Do(func() { close(u.done) })
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
	} else if len(msg) > 7 && msg[:7] == "rename|" { //msg>7个字符 且 前7个字符为“rename|”的情况下
		//修改用户名: 规定用户输入 "rename|用户名" 触发修改
		//1.使用split函数切割字符串提取用户名
		newName := strings.Split(msg, "|")[1] //"|"将字符串切割为2份（数组[0]、[1]）：第0份"rename" 第1份 用户名

		//2.注意要判断用户名是否已经存在，因为使用的是map存储，而name又作为key,所以不能够重复
		_, ok := this.server.OnlineMap[newName] //根据key(name)查询用户，我们不关心值，只关系是否存在
		if ok {
			this.SendMsg("用户名已被使用，请换一个")
		} else {
			//3.修改用户名
			this.server.mapLock.Lock()
			delete(this.server.OnlineMap, this.Name) //先删除OnlineMap的用户
			this.server.OnlineMap[newName] = this    //再添加新名字的用户
			this.server.mapLock.Unlock()
			this.Name = newName //修改this用户的name值
			this.SendMsg("用户名修改成功: " + this.Name + "\n")
		}

	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式：to|张三|消息内容
		//先检查格式
		parts := strings.SplitN(msg, "|", 3) // 最多切 3 段：["to", 姓名, 内容]
		if len(parts) < 3 || parts[1] == "" || parts[2] == "" {
			this.SendMsg("格式不对或内容为空，请使用 to|姓名|内容\n")
			return
		}
		//1.获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		//2.根据用户名得到对方user对象
		this.server.mapLock.RLock()
		remoteUser, ok := this.server.OnlineMap[remoteName]
		this.server.mapLock.RUnlock()
		if !ok { //先判断用户是否存在
			this.SendMsg("该用户不存在！\n")
			return
		}

		//3.获取消息内容，通过对方的user对象将消息内容发送出去
		//如果只有一个"|"则string只会切为2段，只有[0][1]，strings.Split(msg, "|")[2]会报数组越界错误
		conten := strings.Split(msg, "|")[2] //切割出来的第2号元素为消息内容
		remoteUser.SendMsg(this.Name + "对你说：" + conten)

	} else {
		//通过user调用其关联的server中的BroadCast即可
		this.server.BroadCast(this, msg)
	}
}

// 创建一个监听当前对象channel的goroutine方法 用于监听服务端消息并且实时发送给客户端
func (this *User) ListenMessage() {
	for {
		select {
		case msg := <-this.C: //读取C(管道中)的数据
			if _, err := this.conn.Write([]byte(msg + "\n")); err != nil { //将msg写入连接的管道/相当于发送数据
				return // 连接已坏，协程自己退出
			}
		case <-this.done:
			return
		}
	}

}
