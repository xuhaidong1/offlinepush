package domain

// Business 业务名字，域名
type Business struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// Device 设备类型，id
type Device struct {
	Type string `json:"type"`
	ID   string `json:"id"`
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
