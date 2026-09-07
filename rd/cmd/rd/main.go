// wbflow 后端入口：配置加载 → 路由注册 → 优雅退出。
package main

import (
	"fmt"
)

func main() {
	// TODO(huzilin): 按 spec.md §1 落 Gin 路由与依赖注入（handler→service→repo）。
	fmt.Println("wbflow rd backend: skeleton")
}
