# GO语言结构  

## 1. 示例代码  

*更加详细的部分在：* Study/cdoe/main/main.go  

~~~go  
package main

import "fmt"

func main() {
   /* 这是我的第一个简单的程序 */
   fmt.Println("Hello, World!")
}
~~~

## 2. 结构介绍

1. 第一行代码 package main 定义了包名。你必须在源文件中非注释的第一行指明这个文件属于哪个包，如：package main。package main表示一个可独立执行的程序，每个 Go 应用程序都包含一个名为 main 的包。
 
   ### package main 的特殊之处  

   1. 它是程序入口包。只有 package main 会被编译成可执行文件（go build 出二进制、go run 直接跑）；其他包（如 package utils）编译成可被引用的库。

   2. 它不能被 import。写 import "xxx/main" 会报错：import "x" is a program, not an importable package——程序不是库。

   3. 一个目录的包名要统一。E:\GOstudy\hello\ 下所有 .go 文件都必须是 package main（测试文件可写 package main_test），否则报 found packages main and xxx。

   4. 包名和目录名是两回事。目录可以叫 hello，里面写 package main，编译器认的是 package 声明；import 认的却是导入路径（module 路径 + 目录），不是包名。

2. 下一行 import "fmt" 告诉 Go 编译器这个程序需要使用 fmt 包（的函数，或其他元素），fmt 包实现了格式化 IO（输入/输出）的函数。
 
   ### import有3种写法

   ~~~go
   import f "fmt"      // 别名：用 f.Println
   import _ "net/http/pprof"  // 只用它的 init() 副作用，不直接引用
   import . "fmt"      // 省略包名，直接用 Println（不推荐，易冲突）
   ~~~
   
3. 下一行 func main() 是程序开始执行的函数。main 函数是每一个可执行程序所必须包含的，一般来说都是在启动后第一个执行的函数（如果有 init() 函数则会先执行该函数）。   **注意：** ***每个可执行程序必须要有一个main包，而且一个main包必须有且仅有一个 func main() 函数 一一对应***

