package main

import (
	"fmt"
	"net"
	"net/rpc"
	"sync" // sync包提供了互斥锁（Mutex），用于防止多个员工同时抢任务时数据错乱
)

// 定义一个任务结构体
type Task struct {
	Id   int    // 任务编号
	Text string // 要拆分的句子
}

// 员工提交结果的结构体
type Result struct {
	TaskId int      // 对应任务的编号
	Words  []string // 拆分后的单词列表
}

// Master（老板）的结构体
type Master struct {
	mu        sync.Mutex // 互斥锁，保护下面两个字段不被并发写乱
	tasks     []Task     // 待办任务列表
	taskIndex int        // 当前分配到的任务索引（指向下一个要发的任务）
	done      bool       // 是否所有任务都已分配完
}

// RPC方法：员工向老板要任务（老板从队列里弹出一个任务发给它）
func (m *Master) GetTask(req struct{}, reply *Task) error {
	m.mu.Lock()         // 上锁，防止其他员工同时进来抢
	defer m.mu.Unlock() // 函数结束时解锁

	// 如果任务索引超出范围，说明没任务了，返回空任务（Id=0表示结束）
	if m.taskIndex >= len(m.tasks) {
		reply.Id = 0 // 用0作为“结束信号”
		return nil
	}

	// 取出当前任务，并把索引指向下一个
	*reply = m.tasks[m.taskIndex]
	m.taskIndex++

	// 如果取完后发现是最后一个，标记完成
	if m.taskIndex >= len(m.tasks) {
		m.done = true
	}
	return nil
}

// RPC方法：员工提交结果给老板
func (m *Master) SubmitResult(req *Result, reply *struct{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 打印员工返回的结果（在实际MapReduce中会存到磁盘，这里我们先打印出来看效果）
	fmt.Printf("【老板】收到任务 %d 的结果: %v (共 %d 个单词)\n", req.TaskId, req.Words, len(req.Words))
	return nil
}

func main() {
	// 初始化老板，准备3个待办任务（模拟要拆分的3句话）
	master := &Master{
		tasks: []Task{
			{Id: 1, Text: "Hello distributed world"},
			{Id: 2, Text: "Go RPC is very powerful"},
			{Id: 3, Text: "MapReduce is fun"},
		},
		taskIndex: 0,
		done:      false,
	}

	// 注册RPC服务
	err := rpc.Register(master)
	if err != nil {
		fmt.Println("注册RPC服务失败:", err)
		return
	}

	// 启动TCP监听
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println("监听端口失败:", err)
		return
	}
	defer listener.Close()

	fmt.Println("【老板】已启动，有 3 个任务待分配，等待员工上门取活...")

	// 循环接受员工连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// 每个员工独立一个协程处理
		go rpc.ServeConn(conn)
	}
}