package main

import (
	"fmt"
	"net/rpc"
	"strings"
	"time" // time包用于让员工抢活失败时休息一下，避免疯狂空转
)

// 定义结构体（必须和老板那边完全一样）
type Task struct {
	Id   int
	Text string
}

type Result struct {
	TaskId int
	Words  []string
}

func main() {
	// 连接老板
	client, err := rpc.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		fmt.Println("连接老板失败:", err)
		return
	}
	defer client.Close()

	// 给这个员工起个“代号”（用进程ID或随机数，这里简单用固定名字）
	workerName := "员工-A"

	fmt.Printf("【%s】已上线，开始抢活...\n", workerName)

	for {
		// 1. 向老板要任务
		var task Task
		err := client.Call("Master.GetTask", struct{}{}, &task)
		if err != nil {
			fmt.Printf("【%s】要任务失败: %v\n", workerName, err)
			break
		}

		// 2. 如果任务ID为0，说明没活了，下班
		if task.Id == 0 {
			fmt.Printf("【%s】所有任务已完成，下班回家！\n", workerName)
			break
		}

		// 3. 干活：拆分单词
		fmt.Printf("【%s】抢到任务 %d: \"%s\"，开始拆分...\n", workerName, task.Id, task.Text)
		words := strings.Fields(task.Text)
		result := Result{
			TaskId: task.Id,
			Words:  words,
		}

		// 4. 提交结果给老板（假装处理需要花点时间，模拟真实计算）
		time.Sleep(5 * time.Second) // 故意等1秒，让其他员工有机会抢别的任务
		var reply struct{}
		err = client.Call("Master.SubmitResult", &result, &reply)
		if err != nil {
			fmt.Printf("【%s】提交结果失败: %v\n", workerName, err)
		} else {
			fmt.Printf("【%s】任务 %d 提交成功！\n", workerName, task.Id)
		}
	}
}