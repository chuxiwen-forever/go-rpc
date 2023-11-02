package gorpc

import (
	"errors"
	"fmt"
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

// 将参数call添加到client.pending, 更新client.seq
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutDown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// 根据seq, 从client.pending中移除对应到call
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// 服务端和客户端出现错误时, 将shutdown设为true, 将错误信息通知给状态为pending的call
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// 接受消息
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err := client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		// call不存在,
		case call == nil:
			err = client.cc.ReadBody(nil)
		// call不为空, 但是服务端出错
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		// 服务器处理正常, 需要从body中读取Reply的值
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}
