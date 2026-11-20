package main

//一般导入多个包则使用以下方式：improt("..." "...")
import (
	"fmt"
	"time"
)

/*
也可以一个个导入，如
improt "fmt"
improt "time"
*/

// 注意在go语言的语法层面 **一定是要求函数和“左大括号：{ ” 是在同一行的，否则报错：unexpected semicolon or newline before {
// 其他语言都是可以的，只有go语言不可以
func main() {
	//go语言加不加;都可以
	fmt.Println("hello\n")
	/*
			暂停 1 秒；写 time.Sleep(1) 是 1 纳秒，等于不睡
			func Sleep(d Duration)
		    Sleep pauses the current goroutine for at least the duration d.
		    A negative or zero duration causes Sleep to return immediately.
			输入参数为Duration 底层是int64单位是 纳秒
			语义：只暂停当前 goroutine 至少 d，其它 goroutine 照跑；d <= 0 立即返回
			常用单位常量：Nanosecond Microsecond Millisecond Second Minute Hour（没有 Day，一天写 24 * time.Hour）
	*/
	time.Sleep(1 * time.Second)
	fmt.Println("************")
}
