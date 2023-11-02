package gorpc

import (
	"errors"
	"gorpc/codec"
	"io"
	"sync"
)

type Call struct {
	Seq           uint64
	ServiceMethod string      // 方法名字
	Args          interface{} // 方法参数
	Reply         interface{} // 结果
	Error         error       // 出现错误
	Done          chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc       codec.Codec // 编码器
	opt      *Option
	sending  sync.Mutex   // 互斥锁
	header   codec.Header // 请求头
	mu       sync.Mutex
	seq      uint64 // 请求编号
	pending  map[uint64]*Call
	closing  bool // 用户主动关闭
	shutdown bool // 有错误发生
}

var _ io.Closer = (*Client)(nil)

var ErrShutDown = errors.New("connection is shut down")

// Close 关闭链接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutDown
	}
	client.closing = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}
