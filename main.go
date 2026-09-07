// main.go 这是我们的主程序文件
package main // package main 表示这是一个可执行程序，而不是库

import (
	"fmt"      // fmt 是格式化输入输出的库，用来打印
	"strings"  // strings 库里有现成的“按空格拆分单词”的函数
)

// 这是我们今天写的核心函数：拆分单词
// 输入：一段英文文本（string类型）
// 输出：单词列表（[]string类型，即字符串切片）
func splitWords(text string) []string {
	// strings.Fields 会自动按空格、换行、制表符分割，比手动写循环聪明多了
	words := strings.Fields(text)
	return words
}

// 主函数，程序启动时自动执行
func main() {
	// 测试一下我们写的拆分函数
	sentence := "Hello world from Go language"
	result := splitWords(sentence)
	
	// 打印结果到屏幕
	fmt.Println("拆分结果:", result)
	fmt.Println("单词总数:", len(result)) // len() 是Go的内置函数，计算长度
}