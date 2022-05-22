package job

import (
    "context"
    "encoding/json"
    "github.com/gohade/hade/framework/contract"
    "time"
)

type CreateUser struct {
    Name string
}

func (c *CreateUser) MarshalText() (text []byte, err error) {
    return json.Marshal(c)
}

func (c *CreateUser) UnmarshalText(text []byte) error {
    return json.Unmarshal(text, c)
}

func (c *CreateUser) JobName() string {
    return "create_user"
}

func (c *CreateUser) Fire(ctx context.Context, meta *contract.JobMeta) error {
    logger := meta.Container.MustMake(contract.LogKey).(contract.Log)
    logger.Info(ctx, "get create user job: meta", map[string]interface{}{
        "meta": meta,
    })
    return nil
}

func (c *CreateUser) OnConnection() string {
    return "queue.default"
}

func (c *CreateUser) MaxRetry() int {
    return 3
}

func (c *CreateUser) Later() time.Duration {
    return 3 * time.Second
}

func (c *CreateUser) Timeout() time.Duration {
    return 3 * time.Second
}
