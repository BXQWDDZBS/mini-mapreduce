package main

import (
	"fmt"
	"net"
	"net/rpc"   // Go标准库自带的RPC包，负责网络通信
	"strings"
)

// 定义一个结构体，用来包装我们的拆分函数（RPC要求必须用结构体包装）
type Splitter struct{}

// 定义“请求”的结构体：客户端发什么数据过来
type Request struct {
	Text string // 要拆分的句子
}

// 定义“响应”的结构体：服务器返回什么数据
type Response struct {
	Words []string // 拆分后的单词列表
	Count int      // 单词总数
}

// 这是核心方法：拆分单词，并填充到Response里
// 参数：req（请求），res（响应）—— Go的RPC规定，响应必须是指针类型
func (s *Splitter) Split(req Request, res *Response) error {
	// 执行拆分逻辑
	words := strings.Fields(req.Text)
	// 把结果填入响应
	res.Words = words
	res.Count = len(words)
	// 返回nil表示没有错误
	return nil
}

func main() {
	// 创建一个Splitter实例
	splitter := new(Splitter)
	// 把这个实例注册到RPC默认服务器，让客户端能调用它
	err := rpc.Register(splitter)
	if err != nil {
		fmt.Println("注册RPC服务失败:", err)
		return
	}
	// 启动TCP监听，端口号 1234（随便选的，没被占用就行）
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println("监听端口失败:", err)
		return
	}
	defer listener.Close()
	fmt.Println("【老板】已启动，监听端口 1234，等待员工来派活...")

	// 循环接受客户端连接，并交给RPC处理
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// 每个连接在独立的goroutine（轻量级线程）里处理，不阻塞其他员工
		go rpc.ServeConn(conn)
	}
}