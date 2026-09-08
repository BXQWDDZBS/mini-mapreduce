package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os" // 新增：用于读取命令行参数
	"sync"
	"time"
)

// ============================================================================
// 常量定义
// ============================================================================

// 任务状态常量，用于标识任务当前的处理阶段
const (
	Pending    = 0 // 待处理（尚未被领取）
	InProgress = 1 // 处理中（已被某个员工领取，正在执行）
	Completed  = 2 // 已完成（员工已提交结果）
)

// ============================================================================
// 数据结构定义
// ============================================================================

// Task 表示一个待处理的分词任务
// Id：任务唯一编号（从1开始递增）
// Text：需要分词的原始句子
// Status：当前任务状态（Pending / InProgress / Completed）
// StartTime：任务被员工领取的时间，用于超时判断
type Task struct {
	Id        int
	Text      string
	Status    int
	StartTime time.Time
}

// Result 表示员工提交的处理结果
// TaskId：对应任务的编号
// Words：分词后的单词列表（按空格分割的结果）
type Result struct {
	TaskId int
	Words  []string
}

// Master 是“老板”角色的核心结构体，负责管理任务队列、分发任务和接收结果
// mu：互斥锁，保护并发访问 tasks、taskIndex 等共享字段
// tasks：存储所有任务的切片
// taskIndex：用于快速定位下一个待分配任务的索引（优化遍历效率）
// done：预留字段（当前未使用，可用于标记整体完成状态）
type Master struct {
	mu        sync.Mutex
	tasks     []Task
	taskIndex int
	done      bool
}

// ============================================================================
// 后台监控协程（超时恢复机制）
// ============================================================================

// monitor 在后台独立运行，每隔 2 秒检查一次所有正在执行的任务
// 如果某个任务处于 InProgress 状态且持续时间超过 6 秒，则认为该任务超时，
// 将其状态重置为 Pending，并调整 taskIndex 以便该任务能被重新分配。
// 这种机制实现了简单的容错：当员工崩溃或处理过慢时，任务不会被永久阻塞。
func (m *Master) monitor() {
	for {
		time.Sleep(2 * time.Second) // 每 2 秒巡检一次

		m.mu.Lock() // 加锁以安全访问任务列表
		now := time.Now()
		for i := range m.tasks {
			task := &m.tasks[i]
			// 如果任务正在执行且已耗时超过 6 秒
			if task.Status == InProgress && now.Sub(task.StartTime) > 6*time.Second {
				fmt.Printf("【老板巡检】⚠️ 任务 %d 超时未完成（已等待 %v），重新放回队列！\n",
					task.Id, now.Sub(task.StartTime))
				// 重置任务状态为待处理，并清空开始时间
				task.Status = Pending
				task.StartTime = time.Time{}
				// 将 taskIndex 调整为该任务的位置，确保下次 GetTask 能优先分配它
				// 注意：如果 taskIndex 已经大于 i，则回退到 i，否则保持不变
				if m.taskIndex > i {
					m.taskIndex = i
				}
			}
		}
		m.mu.Unlock() // 解锁
	}
}

// ============================================================================
// RPC 服务方法（供员工远程调用）
// ============================================================================

// GetTask 是 RPC 方法，员工通过此方法向老板请求一个待处理的任务。
// 参数 req 为空结构体（占位），reply 用于返回分配到的任务。
// 返回值 error 表示 RPC 调用是否成功。
// 逻辑：从 taskIndex 开始向后查找第一个状态为 Pending 的任务，
// 找到后将其状态改为 InProgress，记录开始时间，并返回该任务。
// 如果找不到任何待处理任务，则返回一个 Id=0 的空任务（员工收到后应等待）。
func (m *Master) GetTask(req struct{}, reply *Task) error {
	m.mu.Lock()         // 加锁，防止并发修改
	defer m.mu.Unlock() // 函数返回时自动解锁

	// 从当前索引开始遍历任务列表
	for i := m.taskIndex; i < len(m.tasks); i++ {
		if m.tasks[i].Status == Pending {
			// 复制任务数据到 reply
			*reply = m.tasks[i]
			// 更新任务状态为处理中，并记录开始时间
			m.tasks[i].Status = InProgress
			m.tasks[i].StartTime = time.Now()
			// 将索引移动到下一个位置，提高后续查找效率
			m.taskIndex = i + 1
			return nil
		}
	}
	// 没有待处理任务，返回空任务（Id=0 表示无任务）
	reply.Id = 0
	return nil
}

// SubmitResult 是 RPC 方法，员工完成分词后调用此方法提交结果。
// req 包含任务 ID 和分词结果，reply 为无返回值（空结构体）。
// 逻辑：根据任务 ID 查找对应任务，将其状态更新为 Completed，
// 并打印结果日志（用于展示）。
func (m *Master) SubmitResult(req *Result, reply *struct{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 遍历任务列表，找到对应 ID 的任务
	for i := range m.tasks {
		if m.tasks[i].Id == req.TaskId {
			// 标记任务为已完成
			m.tasks[i].Status = Completed
			fmt.Printf("【老板】✅ 收到任务 %d 的结果: %v (共 %d 个单词)\n",
				req.TaskId, req.Words, len(req.Words))
			break
		}
	}
	return nil
}

// ============================================================================
// 主程序入口
// ============================================================================

func main() {
	// =========================================================
	// 【核心改动】：支持动态数量的自定义输入句子
	// =========================================================

	// 1. 读取命令行参数（os.Args[0] 是程序名称，[1:] 是用户传入的参数）
	args := os.Args[1:]

	// 2. 如果用户未提供任何句子，给出友好提示，并使用默认的示例句子
	if len(args) == 0 {
		fmt.Println("========================================")
		fmt.Println("【提示】你未输入任何自定义句子。")
		fmt.Println("用法示例: go run server.go \"句子1\" \"句子2\" \"句子3\"")
		fmt.Println("正在使用默认示例句子进行演示...")
		fmt.Println("========================================")
		args = []string{
			"Hello default world",
			"Go RPC is cool",
			"Fault tolerance is important",
		}
	}

	// 3. 根据用户输入的句子数量，动态构建任务列表
	//    每个句子生成一个 Task，ID 从 1 开始递增，初始状态为 Pending
	tasks := make([]Task, len(args))
	for i, text := range args {
		tasks[i] = Task{
			Id:     i + 1, // 编号从 1 开始
			Text:   text,  // 用户输入的原始句子
			Status: Pending,
		}
	}

	// 4. 创建 Master 实例，初始化任务队列和索引
	master := &Master{
		tasks:     tasks,
		taskIndex: 0,
		done:      false,
	}

	// 5. 启动后台监控协程，用于超时检测和任务恢复
	go master.monitor()

	// 6. 注册 Master 的 RPC 服务（使其方法可被远程调用）
	err := rpc.Register(master)
	if err != nil {
		fmt.Println("注册RPC服务失败:", err)
		return
	}

	// 7. 在 TCP 端口 1234 上监听，等待员工连接
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		fmt.Println("监听端口失败:", err)
		return
	}
	defer listener.Close() // 程序退出时关闭监听器

	// 8. 打印启动信息，显示当前加载的任务数量和超时设定
	fmt.Printf("========================================\n")
	fmt.Printf("【老板】已启动，共加载 %d 个任务。\n", len(tasks))
	fmt.Printf("【老板】超时检测时间设定为 6 秒。\n")
	fmt.Printf("【老板】等待员工上门取活...\n")
	fmt.Printf("========================================\n")

	// 9. 主循环：不断接受新的 TCP 连接，并为每个连接启动一个独立的
	//     goroutine 处理 RPC 请求。这样老板可以同时服务多个员工。
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue // 连接出错则忽略，继续等待下一个连接
		}
		// 将连接交给 RPC 框架处理，该调用会阻塞直到连接关闭，
		// 因此放在独立的 goroutine 中，使主循环能继续接受新连接。
		go rpc.ServeConn(conn)
	}
}