package main

import (
	"fmt"
)

// 变量的申明：
/*
4种变量申明方式：
使用方法1.2.3是可以申明全局变量的，但是方法4 := 不可以用于申明全局变量
:= 只能在 函数体内申明
*/

var gA int = 500
var gB = 2.18

//  gC := 200  报错："expected declaration, found gC"

func main() {
	//1.申明一个变量：  var + 变量名 + 变量类型   默认值是0或空字符串“”
	var name string
	var a int
	fmt.Println("a = ", a)
	fmt.Println("name = ", name)

	//2.申明一个变量，并初始化一个值（即初始化），并且和其他语言不一样：是a变量类型后跟=，不是变量后面
	var name2 string = "lzt"
	var b float32 = 3.14
	var c float64 = 3.1415926
	// Println 不做格式化，会原样打印 %f/%s；要格式化用 Printf（和c语言一样printf才是格式化打印）
	fmt.Printf("b = %f \nc = %f \nname2 = %s\n\n", b, c, name2) //Go 的 %f 默认打印 6 位小数
	//用%.7刚好打印c7位小数
	fmt.Printf("c = %.7f\n", c)
	fmt.Printf("c = %.8f\n", c)
	fmt.Printf("c = %.20f\n\n", c)                            //c = 3.14159260000000006841   属于正常精度误差
	fmt.Printf("b = %v\nc = %v\nname2 = %s\n\n", b, c, name2) // 3.14 / 3.1415926
	// 若坚持用 Println：fmt.Println(b, c, name2)
	fmt.Println("pln c = ", c) //fmt.Println 每个参数用的是默认格式 %v；对浮点数来说，%v 等价于 %g，而 %g 的默认精度是“能唯一还原这个值的最短位数”，不是固定的 6 位。

	//3.在初始化时，可以省去变量类型，通过值自动匹配当前的数据类型 （不推荐该写法）
	var d = 100
	fmt.Println("d = ", d)
	//通过格式化打印：Printf() 将变量类型打印出来：使用 %T 占位符
	fmt.Printf("type of \nd : %T\nname : %T\nb : %T\nc : %T\n", d, name, b, c)

	//4.(常用方法)省去var关键字，直接使用 := 自动匹配
	e := 100
	s := "ggboy"
	//如果是自动匹配，则浮点类型会自动使用float64
	fmt.Printf("e = %d \ntype_E: %T\ns = %s\ntype_S: %T\n", e, e, s, s)

	//打印全局变量
	fmt.Printf("gA = %d\ngB = %f\n", gA, gB)

	//5.申明多个变量：
	//同一类型的情况下：
	var x, y int = 3, 3
	fmt.Printf("x = %d; y = %d\n", x, y)
	var x1, y1 = 7, 8
	fmt.Printf("x1 = %d; y1 = %d\n\n", x1, y1)
	//不同类型的情况，（不要加类型关键字）
	var x3, s3 = 7, "abc"
	fmt.Printf("x3 = %d; s3 = %s\n", x3, s3)
	//多行多变量申明
	var (
		v int  = 540
		j bool = true
	)
	fmt.Println("v =", v, "; j =", j, "\n") //Println()的规则是它在相邻操作数之间总是插入一个空格，“这个行为没有开关可以关闭”（不像 Printf 那样可以自己控制格式），无法打印v = 540; j = true 这样的内容:（数字后面紧跟“;”）,只能是v = 540 ; j = true 或v = 540 ;j = true 。要实现只能通过下面方法
	//fmt.Println("v =", strconv.Itoa(v)+";", "j =", j, "\n")  //也可以转为字符串再打印。

	//Sprintf() 将格式化后的字符串返回，和Printf()用法一样，只是把“打印内容”改为“返回内容”
	var stockcode = 124
	var enddate = "2026.11.24"
	var url = "Code = %d\nEndDate = %s\n"
	var target_url = fmt.Sprintf(url, stockcode, enddate)
	fmt.Println(target_url)
}
