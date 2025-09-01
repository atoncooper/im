package utils

// 负载均衡器
// 在本次中只支持一致性Hash -- Ketama 以及简单轮询
// 在后续更新中会持续推出更多负载选择

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgryski/go-ketama"
)

// Balancer 负载均衡器接口
// 定义所有负载均衡器必须实现的方法

type Balancer interface {
	// Balance 根据键选择一个节点
	// key: 用于哈希的键
	// 返回: 选中的节点地址
	Balance(key string) (string, error)

	// UpdateNodes 更新节点列表
	// nodes: 新的节点信息
	// 返回: 错误信息
	UpdateNodes(nodes any) error

	// GetName 获取负载均衡器名称
	GetName() string
}

var (
	// ErrBalancerType 表示负载均衡器类型错误
	ErrBalancerType = errors.New("invalid balancer type")
	// ErrEmptyNodes 表示节点列表为空
	ErrEmptyNodes = errors.New("nodes cannot be empty")
	// ErrNodeNotFound 表示未找到节点
	ErrNodeNotFound = errors.New("no node available")
)

// BalancerFactory 创建负载均衡器的工厂函数
// typ: 负载均衡器类型 ("hash", "roundrobin")
// nodeInfo: 节点信息
// 返回: 负载均衡器实例和错误信息
func NewBalancer(typ string, nodeInfo any) (Balancer, error) {
	switch typ {
	case "hash":
		nodes, ok := nodeInfo.(map[string]int)
		if !ok {
			return nil, fmt.Errorf("hash balancer requires map[string]int node info: %w", ErrBalancerType)
		}
		return NewHashBalancer(nodes)
	case "roundrobin":
		nodes, ok := nodeInfo.([]string)
		if !ok {
			return nil, fmt.Errorf("roundrobin balancer requires []string node info: %w", ErrBalancerType)
		}
		return NewRoundRobinBalancer(nodes)
	default:
		return nil, fmt.Errorf("unsupported balancer type: %s: %w", typ, ErrBalancerType)
	}
}

// HashBalancer 哈希负载均衡器
// 基于 Ketama 算法实现一致性哈希

type HashBalancer struct {
	continuum *ketama.Continuum
	name      string
	mutex     sync.RWMutex // 保护 continuum 字段
}

// NewHashBalancer 创建哈希负载均衡器
// nodes: 节点及其权重的映射
// 返回: 哈希负载均衡器实例和错误信息
func NewHashBalancer(nodes map[string]int) (*HashBalancer, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyNodes
	}

	buckets := make([]ketama.Bucket, 0, len(nodes))
	for node, weight := range nodes {
		buckets = append(buckets, ketama.Bucket{
			Label:  node,
			Weight: weight,
		})
	}

	continuum, err := ketama.New(buckets)
	if err != nil {
		return nil, fmt.Errorf("failed to create ketama continuum: %w", err)
	}

	return &HashBalancer{
		continuum: continuum,
		name:      "hash-balancer",
	}, nil
}

// Balance 基于键进行哈希负载均衡
// key: 用于哈希的键
// 返回: 选中的节点地址和错误信息
func (h *HashBalancer) Balance(key string) (string, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.continuum == nil {
		return "", ErrNodeNotFound
	}

	// 记录选择时间，可用于监控
	startTime := time.Now()
	defer func() {
		// 实际应用中可以记录选择耗时
		// metrics.RecordBalancerLatency(h.name, time.Since(startTime))
		log.Default().Println("[INFO] Balance cost:", time.Since(startTime))
	}()

	node := h.continuum.Hash(key)
	if node == "" {
		return "", ErrNodeNotFound
	}

	return node, nil
}

// UpdateNodes 更新哈希负载均衡器的节点
// nodes: 新的节点及其权重的映射
// 返回: 错误信息
func (h *HashBalancer) UpdateNodes(nodes any) error {
	newNodes, ok := nodes.(map[string]int)
	if !ok {
		return fmt.Errorf("hash balancer requires map[string]int node info: %w", ErrBalancerType)
	}

	if len(newNodes) == 0 {
		return ErrEmptyNodes
	}

	buckets := make([]ketama.Bucket, 0, len(newNodes))
	for node, weight := range newNodes {
		buckets = append(buckets, ketama.Bucket{
			Label:  node,
			Weight: weight,
		})
	}

	newContinuum, err := ketama.New(buckets)
	if err != nil {
		return fmt.Errorf("failed to create new ketama continuum: %w", err)
	}

	h.mutex.Lock()
	h.continuum = newContinuum
	h.mutex.Unlock()

	return nil
}

// GetName 获取负载均衡器名称
func (h *HashBalancer) GetName() string {
	return h.name
}

// RoundRobinBalancer 轮询负载均衡器
// 基于原子操作实现线程安全的轮询

type RoundRobinBalancer struct {
	index int64
	nodes []string
	name  string
	mutex sync.RWMutex // 保护 nodes 字段
}

// NewRoundRobinBalancer 创建轮询负载均衡器
// nodes: 节点列表
// 返回: 轮询负载均衡器实例和错误信息
func NewRoundRobinBalancer(nodes []string) (*RoundRobinBalancer, error) {
	if len(nodes) == 0 {
		return nil, ErrEmptyNodes
	}

	return &RoundRobinBalancer{
		index: 0,
		nodes: append([]string{}, nodes...), // 创建副本避免外部修改
		name:  "roundrobin-balancer",
	}, nil
}

// Balance 进行轮询负载均衡
// key: 未使用，仅为满足接口要求
// 返回: 选中的节点地址和错误信息
func (r *RoundRobinBalancer) Balance(_ string) (string, error) {
	r.mutex.RLock()
	nodesLen := len(r.nodes)
	r.mutex.RUnlock()

	if nodesLen == 0 {
		return "", ErrNodeNotFound
	}

	// 原子操作确保线程安全
	// 使用取模运算避免索引越界
	index := atomic.AddInt64(&r.index, 1) % int64(nodesLen)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// 再次检查节点列表长度，防止在获取索引后节点被更新
	if len(r.nodes) != nodesLen {
		// 如果节点列表已更改，重新获取索引
		index = atomic.AddInt64(&r.index, 1) % int64(len(r.nodes))
	}

	return r.nodes[index], nil
}

// UpdateNodes 更新轮询负载均衡器的节点
// nodes: 新的节点列表
// 返回: 错误信息
func (r *RoundRobinBalancer) UpdateNodes(nodes any) error {
	newNodes, ok := nodes.([]string)
	if !ok {
		return fmt.Errorf("roundrobin balancer requires []string node info: %w", ErrBalancerType)
	}

	if len(newNodes) == 0 {
		return ErrEmptyNodes
	}

	r.mutex.Lock()
	r.nodes = append([]string{}, newNodes...) // 创建副本避免外部修改
	r.mutex.Unlock()

	// 重置索引，避免可能的不均衡
	atomic.StoreInt64(&r.index, 0)

	return nil
}

// GetName 获取负载均衡器名称
func (r *RoundRobinBalancer) GetName() string {
	return r.name
}
