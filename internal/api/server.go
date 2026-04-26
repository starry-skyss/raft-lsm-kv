// internal/api/server.go
package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"raft-lsm-kv/internal/store"
)

// StartGateway 启动对外的统一 API 网关
func StartGateway(kvStores []*store.KVStore, port string) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()        // 创建一个没有任何默认中间件的裸引擎
	r.Use(gin.Recovery()) // 仅挂载 Panic 恢复中间件

	// 自定义仅记录错误的日志中间件
	r.Use(func(c *gin.Context) {
		c.Next() // 先执行具体的请求处理逻辑

		status := c.Writer.Status()
		// 忽略 404 (Not Found)，只打印真正的服务端故障 (如 500) 或 400 坏请求
		if len(c.Errors) > 0 || (status >= 400 && status != http.StatusNotFound) {
			fmt.Printf("[Error] %s | %d | %s | %s\n",
				time.Now().Format("15:04:05"),
				status,
				c.Request.Method,
				c.Request.URL.Path,
			)
		}
	})

	// 1. 处理 PUT 请求
	r.POST("/put", func(c *gin.Context) {
		var req struct {
			Key   string `json:"key" binding:"required"`
			Value string `json:"value" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 轮询 3 个节点，寻找 Leader
		for i, kv := range kvStores {
			err := kv.Put(req.Key, req.Value)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"status": "success", "leader": i})
				return
			}
			if strings.Contains(err.Error(), "not leader") {
				continue // 换下一个试试
			}
			fmt.Printf("❌ [DEBUG] Put 失败，报错内容: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster has no leader right now"})
	})

	// 2. 处理 GET 请求
	r.GET("/get/:key", func(c *gin.Context) {
		key := c.Param("key")

		// 轮询寻找 Leader
		for i, kv := range kvStores {
			val, exists, err := kv.Get(key) // 注意这里 Get 的签名需要修改，加上 error

			if err == nil {
				if !exists {
					c.JSON(http.StatusNotFound, gin.H{"error": "key not found", "leader": i})
					return
				}
				c.JSON(http.StatusOK, gin.H{"value": val, "leader": i})
				return
			}

			if strings.Contains(err.Error(), "not leader") {
				continue // 不是 Leader，换下一个
			}
			fmt.Printf("❌ [DEBUG] Get 失败，报错内容: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster has no leader right now"})
	})

	// 3. 处理 DELETE 请求
	r.DELETE("/delete/:key", func(c *gin.Context) {
		key := c.Param("key")

		for i, kv := range kvStores {
			err := kv.Delete(key)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"status": "deleted", "leader": i})
				return
			}
			if strings.Contains(err.Error(), "not leader") {
				continue
			}
			fmt.Printf("❌ [DEBUG] DELETE 失败，报错内容: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cluster has no leader right now"})
	})

	// 启动 HTTP 服务
	fmt.Printf("API Gateway running on http://localhost%s\n", port)
	if err := r.Run(port); err != nil {
		panic("Failed to start API Gateway: " + err.Error())
	}
}
