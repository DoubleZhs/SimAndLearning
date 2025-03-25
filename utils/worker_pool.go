package utils

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

// WorkerPool 表示一个工作池
type WorkerPool struct {
	jobs    chan func()
	wg      sync.WaitGroup
	workers int
	closed  atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewWorkerPool 创建一个新的工作池
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &WorkerPool{
		jobs:    make(chan func(), workers*2), // 缓冲区大小为工作者数量的2倍
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
	pool.Start()
	return pool
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-p.ctx.Done():
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					job()
				}
			}
		}()
	}
}

// Submit 提交一个任务到工作池
// 如果工作池已关闭，返回false，否则返回true
func (p *WorkerPool) Submit(job func()) bool {
	if p.closed.Load() {
		return false
	}

	select {
	case p.jobs <- job:
		return true
	case <-p.ctx.Done():
		return false
	}
}

// Stop 停止工作池
// 安全地停止所有工作协程并等待它们完成
func (p *WorkerPool) Stop() {
	// 如果已经关闭，直接返回
	if p.closed.Swap(true) {
		return
	}

	// 取消上下文，通知所有工作协程退出
	p.cancel()

	// 关闭通道前确保所有工作协程已退出循环
	close(p.jobs)

	// 等待所有工作协程完成
	p.wg.Wait()
}
