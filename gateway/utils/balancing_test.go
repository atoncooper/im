package utils

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// 测试哈希负载均衡器的基本功能
func TestHashBalancerBasic(t *testing.T) {
	// 准备测试数据
	nodes := map[string]int{
		"127.0.0.1:8080": 1,
		"127.0.0.1:8081": 1,
		"127.0.0.1:8082": 1,
	}

	// 创建新的哈希负载均衡器
	balancer, err := NewHashBalancer(nodes)
	if err != nil {
		t.Fatalf("Failed to create hash balancer: %v", err)
	}

	// 测试一致性哈希功能
	// 同一个键应该总是映射到同一个节点
	key := "user:123"
	firstNode, err := balancer.Balance(key)
	if err != nil {
		t.Fatalf("Failed to balance key %s: %v", key, err)
	}

	// 验证一致性
	for i := 0; i < 10; i++ {
		node, err := balancer.Balance(key)
		if err != nil {
			t.Errorf("Failed to balance key %s: %v", key, err)
		}
		if node != firstNode {
			t.Errorf("Consistency hash failed: expected %s, got %s for key %s", firstNode, node, key)
		}
	}

	// 测试不同键的分布
	results := make(map[string]int)
	for i := 0; i < 100; i++ {
		node, err := balancer.Balance(fmt.Sprintf("user:%d", i))
		if err != nil {
			t.Errorf("Failed to balance key: %v", err)
			continue
		}
		results[node]++
	}

	// 验证所有节点都有请求分布
	for node := range nodes {
		if results[node] == 0 {
			t.Errorf("Node %s received no requests", node)
		}
	}

	// 记录分布结果，便于调试
	for node, count := range results {
		t.Logf("Node %s received %d requests", node, count)
	}
}

// 测试轮询负载均衡器的基本功能
func TestRoundRobinBalancerBasic(t *testing.T) {
	// 准备测试数据
	nodes := []string{
		"127.0.0.1:8080",
		"127.0.0.1:8081",
		"127.0.0.1:8082",
	}

	// 创建新的轮询负载均衡器
	balancer, err := NewRoundRobinBalancer(nodes)
	if err != nil {
		t.Fatalf("Failed to create round-robin balancer: %v", err)
	}

	// 测试轮询顺序
	expectedSequence := nodes
	for i := 0; i < len(nodes)*2; i++ {
		node, err := balancer.Balance("")
		if err != nil {
			t.Fatalf("Failed to balance: %v", err)
		}
		expectedNode := expectedSequence[i%len(nodes)]
		if node != expectedNode {
			t.Errorf("Round-robin failed: expected %s, got %s at index %d", expectedNode, node, i)
		}
	}
}

// 测试工厂方法创建不同类型的负载均衡器
func TestNewBalancerFactory(t *testing.T) {
	// 测试创建哈希负载均衡器
	hashNodes := map[string]int{
		"127.0.0.1:8080": 1,
	}
	balancer1, err := NewBalancer("hash", hashNodes)
	if err != nil {
		t.Fatalf("Failed to create hash balancer via factory: %v", err)
	}
	if _, ok := balancer1.(*HashBalancer); !ok {
		t.Fatalf("Expected HashBalancer type, got %T", balancer1)
	}

	// 测试创建轮询负载均衡器
	roundRobinNodes := []string{"127.0.0.1:8080"}
	balancer2, err := NewBalancer("roundrobin", roundRobinNodes)
	if err != nil {
		t.Fatalf("Failed to create round-robin balancer via factory: %v", err)
	}
	if _, ok := balancer2.(*RoundRobinBalancer); !ok {
		t.Fatalf("Expected RoundRobinBalancer type, got %T", balancer2)
	}

	// 测试错误的负载均衡器类型
	_, err = NewBalancer("unknown", nil)
	if err == nil {
		t.Fatalf("Expected error for unknown balancer type, but got nil")
	}

	// 测试错误的节点数据类型
	_, err = NewBalancer("hash", []string{"invalid"})
	if err == nil {
		t.Fatalf("Expected error for invalid node data type, but got nil")
	}
}

// 测试错误处理功能
func TestBalancerErrorHandling(t *testing.T) {
	// 测试空节点列表
	_, err := NewHashBalancer(map[string]int{})
	if !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("Expected ErrEmptyNodes, got %v", err)
	}

	_, err = NewRoundRobinBalancer([]string{})
	if !errors.Is(err, ErrEmptyNodes) {
		t.Errorf("Expected ErrEmptyNodes, got %v", err)
	}

	// 测试更新节点时的错误处理
	hashBalancer, _ := NewHashBalancer(map[string]int{"127.0.0.1:8080": 1})
	err = hashBalancer.UpdateNodes([]string{"invalid"})
	if err == nil {
		t.Errorf("Expected error when updating hash balancer with invalid node type")
	}

	roundRobinBalancer, _ := NewRoundRobinBalancer([]string{"127.0.0.1:8080"})
	err = roundRobinBalancer.UpdateNodes(map[string]int{"invalid": 1})
	if err == nil {
		t.Errorf("Expected error when updating round-robin balancer with invalid node type")
	}
}

// 测试节点更新功能
func TestBalancerUpdateNodes(t *testing.T) {
	// 测试哈希负载均衡器的节点更新
	originalNodes := map[string]int{
		"127.0.0.1:8080": 1,
		"127.0.0.1:8081": 1,
	}
	hashBalancer, _ := NewHashBalancer(originalNodes)

	// 获取更新前的一个键的映射
	key := "user:test"
	oldNode, _ := hashBalancer.Balance(key)

	// 更新节点
	newNodes := map[string]int{
		"127.0.0.1:8082": 1,
		"127.0.0.1:8083": 1,
	}
	err := hashBalancer.UpdateNodes(newNodes)
	if err != nil {
		t.Fatalf("Failed to update hash balancer nodes: %v", err)
	}

	// 验证更新后的节点映射已变化
	newNode, _ := hashBalancer.Balance(key)
	if oldNode == newNode {
		t.Errorf("Node mapping didn't change after update: %s", oldNode)
	}

	// 测试轮询负载均衡器的节点更新
	roundRobinBalancer, _ := NewRoundRobinBalancer([]string{"127.0.0.1:8080"})
	err = roundRobinBalancer.UpdateNodes([]string{"127.0.0.1:8082", "127.0.0.1:8083"})
	if err != nil {
		t.Fatalf("Failed to update round-robin balancer nodes: %v", err)
	}

	// 验证更新后的轮询顺序
	for i := 0; i < 4; i++ {
		node, _ := roundRobinBalancer.Balance("")
		expectedNode := []string{"127.0.0.1:8082", "127.0.0.1:8083"}[i%2]
		if node != expectedNode {
			t.Errorf("Round-robin sequence incorrect after update: expected %s, got %s", expectedNode, node)
		}
	}
}

// 测试并发安全性
func TestBalancerConcurrency(t *testing.T) {
	// 准备测试数据
	nodes := []string{
		"127.0.0.1:8080",
		"127.0.0.1:8081",
		"127.0.0.1:8082",
	}

	// 创建轮询负载均衡器
	balancer, _ := NewRoundRobinBalancer(nodes)

	// 并发请求数量
	concurrency := 100
	requestsPerGoroutine := 100

	// 用于等待所有goroutine完成
	var wg sync.WaitGroup

	// 用于保护结果计数器
	var mu sync.Mutex
	results := make(map[string]int)

	// 启动并发请求
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				node, err := balancer.Balance("")
				if err != nil {
					t.Errorf("Error in concurrent request: %v", err)
					continue
				}
				mu.Lock()
				results[node]++
				mu.Unlock()

				// 随机延迟，增加并发竞争条件
				time.Sleep(time.Duration(rand.Intn(5)) * time.Microsecond)
			}
		}()
	}

	// 等待所有请求完成
	wg.Wait()

	// 验证所有节点都有请求分布
	expectedRequests := concurrency * requestsPerGoroutine / len(nodes)
	for _, node := range nodes {
		count := results[node]
		t.Logf("Node %s received %d requests", node, count)
		// 允许一定的误差范围
		diff := abs(count - expectedRequests)
		if diff > expectedRequests/10 { // 允许10%的误差
			t.Errorf("Node %s request distribution is uneven: got %d, expected around %d", node, count, expectedRequests)
		}
	}
}

// 辅助函数：计算绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
