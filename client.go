package main

import (
	"fmt"
	"net/rpc"
)

// 补上这两个结构体定义（和 server.go 里的一模一样）
type Request struct {
	Text string
}

type Response struct {
	Words []string
	Count int
}

func main() {
	// 连接老板（服务器），地址是本地（127.0.0.1），端口1234
	client, err := rpc.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		fmt.Println("连接老板失败:", err)
		return
	}
	defer client.Close()

	// 构造要发送的请求
	req := Request{Text: "Go RPC is powerful and easy"}
	
	// 创建空响应，等着接收结果
	var res Response

	// 调用远程方法："Splitter.Split"（结构体名.方法名）
	err = client.Call("Splitter.Split", req, &res)
	if err != nil {
		fmt.Println("调用远程拆分失败:", err)
		return
	}

	// 打印老板返回的结果
	fmt.Println("【员工】收到老板返回的拆分结果:", res.Words)
	fmt.Println("【员工】单词总数:", res.Count)
}