package ctxwatch

import (
	"context"
	"sync"
)

// ContextWatcher watches a context and performs an action when the context is canceled. It can watch one context at a
// time.

// ContextWatcher 观察一个 context，并在 context 被取消时执行一个操作。
// 它一次只能观察一个 context。
type ContextWatcher struct {
	onCancel             func()
	onUnwatchAfterCancel func()
	unwatchChan          chan struct{}

	lock              sync.Mutex
	watchInProgress   bool
	onCancelWasCalled bool
}

// NewContextWatcher returns a ContextWatcher. onCancel will be called when a watched context is canceled.
// OnUnwatchAfterCancel will be called when Unwatch is called and the watched context had already been canceled and
// onCancel called.

// NewContextWatcher 返回一个 ContextWatcher。但被观察后的 context 被取消时，会调用 onCancel 函数。
// 但 Unwatch 被调用时，OnUnwatchAfterCancel 会被调用，前提是被观察的 context 已经被取消并且 onCancel 已经被调用。
func NewContextWatcher(onCancel func(), onUnwatchAfterCancel func()) *ContextWatcher {
	cw := &ContextWatcher{
		onCancel:             onCancel,
		onUnwatchAfterCancel: onUnwatchAfterCancel,
		unwatchChan:          make(chan struct{}),
	}

	return cw
}

// Watch starts watching ctx. If ctx is canceled then the onCancel function passed to NewContextWatcher will be called.

// Watch 开始观察 ctx。如果 ctx 被取消，那么传递给 NewContextWatcher 的 onCancel 函数将被调用。
func (cw *ContextWatcher) Watch(ctx context.Context) {
	// 加锁
	cw.lock.Lock()
	defer cw.lock.Unlock()

	// 如果正在观察，则 panic
	if cw.watchInProgress {
		panic("Watch already in progress")
	}

	// 重置 onCancelWasCalled 标志
	cw.onCancelWasCalled = false

	if ctx.Done() != nil { // 如果 ctx 可被取消，这进入后续流程，避免对不可取消的 context 产生额外的 goroutine 开销
		cw.watchInProgress = true // 设置正在观察的标志
		go func() {
			select {
			case <-ctx.Done(): // 当先收到 ctx 的取消信号时，调用 onCancel()，并设置 onCancelWasCalled 标志
				cw.onCancel()
				cw.onCancelWasCalled = true
				// 等待 Unwatch 调用，以便释放 goroutine，避免泄漏
				<-cw.unwatchChan
			case <-cw.unwatchChan: // 当在 context 取消之前先收到 unwatchChan 的信号时，直接返回，不调用 onCancel(),避免 goroutine 泄漏
			}
		}()
	} else { // ctx 不可被取消，则表示该 ContextWatcher 无需观察 ctx
		cw.watchInProgress = false
	}
}

// Unwatch stops watching the previously watched context. If the onCancel function passed to NewContextWatcher was
// called then onUnwatchAfterCancel will also be called.

// Unwatch 停止之前观察的 context。如果传递给 NewContextWatcher 的 onCancel 函数被调用了，那么 onUnwatchAfterCancel 也将被调用。
func (cw *ContextWatcher) Unwatch() {
	cw.lock.Lock()
	defer cw.lock.Unlock()

	if cw.watchInProgress { // 正在观察 context
		// 释放 Watch 中 goroutine，避免泄漏
		cw.unwatchChan <- struct{}{}
		// 当调用过了 onCancel()，则调用 onUnwatchAfterCancel()
		if cw.onCancelWasCalled {
			cw.onUnwatchAfterCancel()
		}

		// 重置正在观察的标志
		cw.watchInProgress = false
	}
}
