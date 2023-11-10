package domain

// Business 业务名字，域名
type Business struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// Message 消息定义
type Message struct {
	Business Business `json:"business"`
	Device   Device   `json:"device"`
}

// Task 生产者生产的消息列表
type Task struct {
	MessageList []Message
	Qps         int
}

type Command struct {
	Type    string
	Content string
}

// PushCommand 生成的推送指令
type PushCommand struct {
	DeviceID string
	Cmd      Command
}

func IsEOF(msg Message) bool {
	return msg.Device.Type == "EOF"
}

// GetEOF EOF消息，用于标识为消息末尾，
// 在并发阻塞队列中无法通过队列是否为空来判断后面是否还有消息
func GetEOF(biz string) Message {
	return Message{
		Business: Business{Name: biz},
		Device:   Device{Type: "EOF"},
	}
}
