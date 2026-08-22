package agent

import (
	"sync"
	"unicode/utf8"

	"github.com/gohade/hade/framework/contract"
)

// settleReasonMaxBytes 是补偿型 tool 消息 Content 的字节上限。
// 历史配额按这个值为每个未闭环的 tool_call 预留，保证补偿写入永不突破总上限。
const settleReasonMaxBytes = 32

const (
	settleReasonCanceled = "canceled"
	settleReasonInternal = "internal error"
)

// 补偿文案必须放得进 settleReasonMaxBytes 预留的配额，否则会被 truncate 截成残缺文案。
// 下面的数组长度是常量表达式：任何一条文案超长都会让编译直接失败。
var (
	_ [settleReasonMaxBytes - len(settleReasonCanceled)]struct{}
	_ [settleReasonMaxBytes - len(settleReasonInternal)]struct{}
)

// memorySession 是单个 Session 的内存存储。
//
// 锁被显式拆成两把，避免读接口被长耗时的 Run 阻塞：
//   - runMu 用 TryLock 保护"一个 Session 同时只能跑一轮 Run"，持有时间与整轮 ReAct 等长；
//   - dataMu 只保护消息切片与字节计数，临界区都是常数级操作。
type memorySession struct {
	id     string
	runMu  sync.Mutex
	dataMu sync.Mutex

	messages []contract.Message
	bytes    int
}

func newMemorySession(id string) *memorySession {
	return &memorySession{id: id}
}

// snapshot 返回消息的深拷贝，调用方可以安全地长期持有与修改。
func (s *memorySession) snapshot() []contract.Message {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	return cloneMessages(s.messages)
}

// length 返回当前消息条数，用于历史超限时回滚。
func (s *memorySession) length() int {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	return len(s.messages)
}

// usedBytes 返回历史已占用的字节数。
func (s *memorySession) usedBytes() int {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	return s.bytes
}

// appendWithin 在"已用字节 + 本次写入 + reserve 预留"不超过 limit 时追加消息。
// reserve 是尚未闭环的 tool_call 的补偿配额，先占住才能保证后续补偿一定写得下。
func (s *memorySession) appendWithin(limit, reserve int, msgs ...contract.Message) error {
	added := 0
	for _, message := range msgs {
		added += messageBytes(message)
	}
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	if s.bytes+added+reserve > limit {
		return contract.ErrHistoryLimit
	}
	s.messages = append(s.messages, cloneMessages(msgs)...)
	s.bytes += added
	return nil
}

// appendReserved 写入已被 appendWithin 预留过配额的补偿消息，因此不再判断上限。
func (s *memorySession) appendReserved(msgs ...contract.Message) {
	added := 0
	for _, message := range msgs {
		added += messageBytes(message)
	}
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	s.messages = append(s.messages, cloneMessages(msgs)...)
	s.bytes += added
}

// truncateTo 回滚到指定条数，用于撤销一组未闭环的 assistant tool_calls 及其 tool 回复。
func (s *memorySession) truncateTo(n int) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	if n < 0 || n >= len(s.messages) {
		return
	}
	for _, message := range s.messages[n:] {
		s.bytes -= messageBytes(message)
	}
	s.messages = s.messages[:n:n]
	if s.bytes < 0 {
		s.bytes = 0
	}
}

func cloneMessages(in []contract.Message) []contract.Message {
	if in == nil {
		return nil
	}
	out := make([]contract.Message, len(in))
	for i, message := range in {
		out[i] = message
		if message.ToolCalls != nil {
			out[i].ToolCalls = append([]contract.ToolCall(nil), message.ToolCalls...)
		}
	}
	return out
}

// messageBytes 统计一条消息占用的主要字符串字节数。
func messageBytes(m contract.Message) int {
	total := len(m.Role) + len(m.Content) + len(m.ToolCallID)
	for _, call := range m.ToolCalls {
		total += len(call.ID) + len(call.Name) + len(call.Arguments)
	}
	return total
}

// settleReserveBytes 返回一组 tool_call 全部走补偿路径时的最坏字节开销。
func settleReserveBytes(calls []contract.ToolCall) int {
	total := 0
	for _, call := range calls {
		total += len("tool") + len(call.ID) + settleReasonMaxBytes
	}
	return total
}

// truncateMessageForRead 裁剪对外暴露的消息内容。
// tool arguments 同样可能很大（模型偶尔会灌进整段调试上下文），必须一起裁。
func truncateMessageForRead(m contract.Message, limit int) contract.Message {
	m.Content = truncate(m.Content, limit)
	for i := range m.ToolCalls {
		m.ToolCalls[i].Arguments = truncate(m.ToolCalls[i].Arguments, limit)
	}
	return m
}

// truncate 按字节裁剪字符串，并保证不切断 UTF-8 多字节字符。
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// deepCopyParameters 深拷贝 JSON Schema，避免注册方与 Agent 共享可变 map。
func deepCopyParameters(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		out[key] = deepCopyValue(value)
	}
	return out
}

func deepCopyValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return deepCopyParameters(typed)
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, item := range typed {
			out[i] = deepCopyValue(item)
		}
		return out
	default:
		return value
	}
}
