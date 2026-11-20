# go语言变量

声明变量一般使用var关键字：  

~~~go
var identifier type
~~~

可以一次声明多个变量:  

~~~go
var identifier1, identifier2 type
~~~

~~~go
package main
import "fmt"
func main() {
    var a string = "Runoob"
    fmt.Println(a)

    var b, c int = 1, 2
    fmt.Println(b, c)
}
~~~