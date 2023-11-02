package day2_client

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
