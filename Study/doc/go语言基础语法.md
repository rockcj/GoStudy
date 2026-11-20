# GO语言基础语法

## go标记

Go程序可以由多个标记组成，可以是关键字、标识符、常量、字符串、符好。....

## 行分隔符

在Go中：一行代表一个语句结束。不需要像其他C类语言一样以分号‘;’结尾。因为这些工作都将由 Go 编译器自动完成。

如果你打算将多个语句写在同一行，它们则必须使用 ; 人为区分，但在实际开发中我们并不鼓励这种做法。

## 注释

和其他语言一样

## 标识符

标识符用来命名变量、类型等程序实体。一个标识符实际上就是一个或是多个字母(A~Z和a~z)数字(0~9)、下划线_组成的序列，但是第一个字符必须是字母或下划线而不能是数字。

**例如:**

~~~go
//合法标识符
mahesh   kumar   abc   move_name   a_123
myname50   _temp   j   a23b9   retVal
~~~

~~~go
//非法标识符
1ab (以数字开头)
case (go语言关键字)
a+b (运算符不允许出现在标识符中)
~~~

## 字符串连接

### Go：用 `+`，但两边必须都是 string

Go 和 Java 一样可以用 `+` 直接拼接字符串：

~~~go
package main

import "fmt"

func main() {
    fmt.Println("Google" + "Runoob") // GoogleRunoob
}
~~~

和 Java 不同的是，**Go 不做隐式类型转换**，`+` 只认 `string + string`：

~~~go
s := "a" + 1                 // ❌ mismatched types string and untyped int
s2 := "a" + strconv.Itoa(1)  // ✅
s3 := fmt.Sprintf("a%d", 1)  // ✅
~~~

⚠️ Go 的字符串**不可变**，`s += x` 每次都会分配新内存，循环里是 O(n²)：

~~~go
var b strings.Builder
for i := 0; i < n; i++ {
    b.WriteString(strconv.Itoa(i))
}
s := b.String()                          // 推荐
strings.Join([]string{"a", "b"}, "-")    // 已知元素时最简洁
~~~

***但是要注意：***

1. **C 语言中完全不可以使用 + 拼接**

   **C：`+` 是指针算术，不是拼接**

   C 没有字符串类型，`"Hello"` 是 `const char[6]`，用的时候退化成 `const char*`（指针）。两个指针相加在 C 里是非法的：

   ~~~c
   char *s = "Hello" + "World";   // ❌ error: invalid operands to binary +
   ~~~

   更危险的是 **指针 + 整数是合法的**，它做的是地址偏移，而不是拼接：

   ~~~c
   char *t = "Hello" + 1;    // ✅ 能编译，结果是 "ello"
   char *u = "Hello" + 40;   // ✅ 也能编译，越界，运行期可能崩溃
   ~~~

   C 只能老老实实用格式化或字符串函数：

   ~~~c
   char buf[64];
   snprintf(buf, sizeof buf, "%s %s", "Hello", "World");  // 推荐，安全

   char *p = malloc(64);
   strcpy(p, "Hello");
   strcat(p, " World");      // 自己保证空间够，溢出就是未定义行为
   ~~~

2. **C++ 的 `+` 是重载出来的，至少一端要是 `std::string`**

   ~~~cpp
   std::string a = "Hello";
   auto s = a + " " + "World";               // ✅ 有 std::string 参与
   auto t = std::string("Hello") + "World";  // ✅

   auto bad = "Hello" + "World";             // ❌ 还是两个 const char*，同 C
   ~~~

   链式 `+` 会产生临时对象，拼得多时改用 `+=` / `append`：

   ~~~cpp
   std::string s;
   s += "Hello";
   s.append(" World");
   ~~~

3. **Java 的 `+` 最宽松**

   只要有一边是 String，另一边会自动 `String.valueOf`，数字、布尔都能直接拼：

   ~~~java
   String s = "Hello" + " " + "World";   // ✅
   String t = "a" + 1 + 2.5 + true;      // ✅ → "a12.5true"
   ~~~

   编译期全是常量时会被折叠成一个字面量。循环里同样别用 `+`：

   ~~~java
   StringBuilder sb = new StringBuilder();
   for (int i = 0; i < n; i++) sb.append(i);
   String r = sb.toString();

   String.join("-", "a", "b");           // 现代写法
   ~~~

### 四种语言对照

| 语言 | `"a" + "b"` | `"a" + 1` | 惯用拼接方式 |
| --- | --- | --- | --- |
| C | ❌ 编译错误 | ⚠️ 能编译，是**指针偏移**（灾难） | `snprintf` / `strcat` |
| C++ | ❌ 两个 `const char*` 报错 | ⚠️ 同上，指针偏移 | `std::string` 的 `+` / `+=` / `append` |
| Java | ✅ | ✅ 自动转字符串 | `+`（少量）/ `StringBuilder` / `String.join` |
| Go | ✅ | ❌ 类型不匹配 | `+`（少量）/ `strings.Builder` / `strings.Join` / `Sprintf` |

### 三条实践结论

1. **C 里永远不要对字符串用 `+`** —— 它不报错的时候比报错更危险。
2. **循环里拼接，四种语言都别用 `+` / `+=`**：C 用 `snprintf` 并注意长度，C++ 用 `append`（有 SSO 和均摊扩容，问题最小），Java 用 `StringBuilder`，Go 用 `strings.Builder`。
3. **Go 的 `+` 最死板**（必须 string + string），但这是它杜绝隐式转换的一贯风格；日常主力是 `fmt.Sprintf` 和 `strings.Builder`。  

## 关键字

1. go中的25个关键字或保留字  

    |  |  |  |  |  |
    | --- | --- | --- | --- | --- |
    | break | default | func | interface | select |
    | case | defer | go | map | struct |
    | chan | else | goto | package | switch |
    | const | fallthrough | if | range | type |
    | continue | for | import | return | var |

2. 除了以上介绍的这些关键字，go语言还有36个预定义标识符:

    |  |  |  |  |  |  |  |  |  |
    | --- | --- | --- | --- | --- | --- | --- | --- | --- |
    | append | bool | byte | cap | close | complex | complex64 | complex128 | uint16 |
    | copy | false | float32 | float64 | imag | int | int8 | int16 | unt32 |
    | int32 | int64 | iota | len | make | new | nil | panic | uint64 |
    | print | println | real | recover | string | true | uint | uint8 | uinttptr |

## 格式化字符串

go语言中使用 `fmt.Sprintf()` / `fmt.Printf()` 格式化字符串并赋值

- Sprintf根据格式化参数生成格式化的字符串并**返回该字符串**
- Printf根据格式化参数生成格式化的字符串并并**写入标准输出**

~~~go
//Sprintf()示例
package main

import (
    "fmt"
)

func main() {
   // %d 表示整型数字，%s 表示字符串
    var stockcode=123     //stockcode是股票代码
    var enddate="2020-12-31"
    var url="Code=%d&endDate=%s"
    var target_url=fmt.Sprintf(url,stockcode,enddate)  //目标url (即目标字符串)
    fmt.Println(target_url)
}
~~~

*输出*:

~~~cmd
Code=123&endDate=2020-12-31
~~~

~~~go
//Print示例:
package main

import (
    "fmt"
)

func main() {
   // %d 表示整型数字，%s 表示字符串
    var stockcode=123
    var enddate="2020-12-31"
    var url="Code=%d&endDate=%s"
    fmt.Printf(url,stockcode,enddate)
}
~~~

*输出*:

~~~cmd
Code=123&endDate=2020-12-31
~~~
