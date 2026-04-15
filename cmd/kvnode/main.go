package main

import (
	"fmt"

	"os"
	"time"

	//"raft-lsm-kv/internal/lsm"
	"raft-lsm-kv/internal/api"
	"raft-lsm-kv/internal/raft"
	"raft-lsm-kv/internal/raft/labgob"
	"raft-lsm-kv/internal/raft/labrpc"
	"raft-lsm-kv/internal/raft/raftapi"
	"raft-lsm-kv/internal/store"
)

func main() {
	labgob.Register(store.Op{})
	net := labrpc.MakeNetwork()
	defer net.Cleanup()

	serverCount := 3
	kvStores := make([]*store.KVStore, serverCount)
	// 1. 提前创建好 3 个空的 RPC Server，并挂载到网络上
	servers := make([]*labrpc.Server, serverCount)
	for i := 0; i < serverCount; i++ {
		serverName := fmt.Sprintf("server-%d", i)
		servers[i] = labrpc.MakeServer()
		net.AddServer(serverName, servers[i])
	}

	for i := 0; i < serverCount; i++ {
		nodeDir := fmt.Sprintf("./data/node_%d", i)
		os.MkdirAll(nodeDir, 0755)
		// 3. 初始化 Raft 的物理持久化器
		persister := raft.MakePersister(nodeDir + "/raft_state.bin")

		// 4. 连接网络端点 (告诉当前节点，其他兄弟是谁)
		peers := make([]*labrpc.ClientEnd, serverCount)
		for j := 0; j < serverCount; j++ {
			if j != i {
				// 注意：在实际的 labrpc 测试中，需要更复杂的网络连线
				// 这里做了简化，让端点互相可见
				clientName := fmt.Sprintf("client-%d-to-%d", i, j)
				peers[j] = net.MakeEnd(clientName)
				net.Connect(clientName, fmt.Sprintf("server-%d", j))
				net.Enable(clientName, true)
			}
		}

		applyCh := make(chan raftapi.ApplyMsg)
		rf := raft.Make(peers, i, persister, applyCh)

		// 【🔥 最关键的连线步骤】：将 Raft 对象注册为 RPC 服务
		// 只有加了这一步，底层的 labrpc 才知道怎么把 RequestVote 路由给 rf
		svc := labrpc.MakeService(rf)
		servers[i].AddService(svc)

		kvStores[i] = store.NewKVStore(nodeDir, rf, applyCh)
	}

	fmt.Println("Cluster starting, waiting for leader election...")
	time.Sleep(3 * time.Second)
	fmt.Println("Cluster ready!")

	// 【修改点】：直接调用 API 层的启动函数，整个世界的运转交给了这一行！
	api.StartGateway(kvStores, ":8080")
}
