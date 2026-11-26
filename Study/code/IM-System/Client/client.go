package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

type Client struct {
	ServerIp   string
	ServerPort int
	Name       string
	conn       net.Conn
	flag       int //当前client的模式
}

// 创建客户端对象
func NewClient(serverIp string, serverPort int) *Client {
	//1.创建对象,初始化赋值
	client := &Client{
		ServerIp:   serverIp,
		ServerPort: serverPort,
		flag:       999,
	}
	//2.连接Server
	addr := fmt.Sprintf("%s:%d", serverIp, serverPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("连接错误:net.Dial: ", err)
		return nil //返回空
	}
	client.conn = conn
	return client
}

// 处理server回应的消息，直接显示标准输出即可
func (c *Client) DealResponse() {
	//一旦c.conn有消息就马上copy到标准输出stdout上
	io.Copy(os.Stdout, c.conn) //该io方法是永久阻塞监听的
}

// 全局变量
var serverIp string
var serverPort int

// go语言中每个文件都会有一个init函数，会在main函数之前去执行
func init() {
	// flag.StringVar 四个参数说明：
	//  1. p      —— 保存结果的变量指针：这里传 &serverIp，解析后用户输入的值会写进 serverIp
	//  2. name   —— 命令行参数名：这里叫 "ip"，启动时用 -ip 192.168.1.10 或 -ip=192.168.1.10 指定
	//  3. value  —— 默认值：用户没写 -ip 时就用 "127.0.0.1"
	//  4. usage  —— 帮助说明：运行 -h / --help 时显示的提示文字
	//
	//  这里的参数 1 &serverIp：必须传指针，因为 StringVar 要把解析出来的值写回去。
	// 	所以 serverIp 得是包级变量（var serverIp string），不能是 main 里的局部变量，否则 init 里拿不到。
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器ip地址(默认为127.0.0.1)") //解析命令行
	flag.IntVar(&serverPort, "port", 8888, "设置服务器端口(默认8888)")
}

// 公聊模式
func (c *Client) PublicChat() {
	fmt.Println(">>>>>请输入聊天内容，exit表示退出该模式")
	var chatMsg string
	fmt.Scanln(&chatMsg)
	for chatMsg != "exit" {
		//消息不为空则发送
		if len(chatMsg) != 0 {
			sendMsg := chatMsg + "\n"
			_, err := c.conn.Write([]byte(sendMsg))
			if err != nil {
				fmt.Println("conn Write:", err)
				break
			}
		}
		//然后要紧接着重置chatMsg，因为要靠其判断
		chatMsg = ""
		fmt.Println(">>>>>请输入聊天内容，exit表示退出该模式")
		fmt.Scanln(&chatMsg) //然后继续输入
	}
}

// 更新用户名
func (c *Client) UpdatedName() bool {
	fmt.Println(">>>>>>>>请输入用户名:")
	fmt.Scanln(&c.Name)
	sendMsg := "rename|" + c.Name + "\n"
	_, err := c.conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("conn.Write:", err)
		return false
	}
	return true
}

// 菜单显示
func (client *Client) menu() bool {
	var flag int
	fmt.Println("",
		"1.公聊模式\n",
		"2.私聊模式\n",
		"3.更新用户名\n",
		"0.退出")

	fmt.Scanln(&flag)

	if flag >= 0 && flag <= 3 {
		client.flag = flag
		return true
	} else {
		fmt.Println(">>>>>>请输入合法范围内的数字<<<<<")
		return false
	}
}

// 业务处理
func (c *Client) run() {
	for c.flag != 0 {
		for !c.menu() { //如果menu不为true则进入死循环，一直调用菜单显示，如果为true则跳出循环，继续执行任务
		}

		//根据不同模式处理不同业务
		switch c.flag {
		case 1:
			//公聊
			fmt.Println("公聊")
			break
		case 2:
			//私聊
			fmt.Println("私聊")
			break
		case 3:
			//更新用户名
			c.UpdatedName()
			break
		}
	}
}

func main() {
	//自动解析命令行
	flag.Parse()
	//连接服务端
	client := NewClient(serverIp, serverPort)
	if client == nil {
		fmt.Println(">>>>>服务器连接失败......")
		return
	}
	fmt.Println(">>>>>>服务器连接成功！")

	//单独开启一个goroutine去处理server的回执消息
	go client.DealResponse() //主要用于接收

	//启动客户端的业务
	client.run() //主要用于发送
}
