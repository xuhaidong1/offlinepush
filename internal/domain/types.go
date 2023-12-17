package domain

// Topic 业务名字，域名
type Topic struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// Message 消息定义
type Message struct {
	Topic  Topic  `json:"topic"`
	Device Device `json:"device"`
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
