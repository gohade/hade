package contract

import (
	"context"
	"encoding"
	"github.com/gohade/hade/framework"
	"time"
)

const QueueKey = "hade:queue"

// Job 元素必须可文本化
type Job interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler

	// JobName job的名称，必须填写
	JobName() string

	// Fire 表示执行一个任务的行为，如果返回error，请谨慎注意是否需要设置MAXRetry
	Fire(cxt context.Context, meta *JobMeta) error
}

// JobMeta 代表当前任务的元数据
type JobMeta struct {
	JobName  string
	JobId    string
	PushTime time.Time
	PopTime  time.Time

	Connection string
	CurrRetry  int
	Container  framework.Container // 必须不为空
}

// OnConnection 如果元素制定了connection配置
// 可以使用不同的connection队列，否则使用queue.default
type OnConnection interface {
	OnConnection() string
}

// NeedMaxRetry 最大尝试次数
type NeedMaxRetry interface {
	MaxRetry() int
}

// NeedTimeout 最大超时时长
type NeedTimeout interface {
	TimeOut() time.Duration
}

// NeedLater 这个任务是否是延迟执行，如果是的话，则延迟执行
type NeedLater interface {
	Later() time.Duration
}

// QueueService 队列服务
type QueueService interface {
	// Register 注册一个元素, 生产者消费者都需要注册元素
	// 对于生产者，未注册的元素不会进入队列
	// 对于消费者，未注册的元素不会进行消费，如果pop出来了，则重新push进入队列
	Register(job Job)
	// Listen 开始监听，根据注册的job有多少connection，启动多少个监听通道
	Listen() error

	// Push 推送元素进入一个队列
	Push(ctx context.Context, job Job) error
	// GoPush 异步push，保证线程安全
	GoPush(ctx context.Context, job Job)
}
