package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gohade/hade/framework/contract"
	"github.com/google/uuid"
)

const (
	sessionsSetKey = "hade:agent:sessions"
	lockTTLDefault = 60 * time.Second
	renewDefault   = 20 * time.Second
)

const luaUnlock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`

const luaPersist = `
if redis.call("GET", KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call("SET", KEYS[2], ARGV[2])
return 1
`

const luaRenew = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`

func sessionKey(id string) string { return "hade:agent:session:" + id }
func lockKey(id string) string    { return "hade:agent:lock:" + id }

type sessionDoc struct {
	Messages []contract.Message `json:"messages"`
	Bytes    int                `json:"bytes"`
}

type redisStore struct {
	client     *redis.Client
	lockTTL    time.Duration
	renewEvery time.Duration
}

func newRedisStore(client *redis.Client) SessionStore {
	return newRedisStoreWithLock(client, lockTTLDefault, renewDefault)
}

func newRedisStoreWithLock(client *redis.Client, lockTTL, renewEvery time.Duration) SessionStore {
	if lockTTL <= 0 {
		lockTTL = lockTTLDefault
	}
	return &redisStore{
		client:     client,
		lockTTL:    lockTTL,
		renewEvery: renewEvery,
	}
}

func (s *redisStore) Create(ctx context.Context) (string, error) {
	id := uuid.New().String()
	payload, err := json.Marshal(sessionDoc{Messages: []contract.Message{}})
	if err != nil {
		return "", contract.ErrInternal
	}
	ok, err := s.client.SetNX(ctx, sessionKey(id), payload, 0).Result()
	if err != nil {
		return "", contract.ErrInternal
	}
	if !ok {
		return "", contract.ErrInternal
	}
	if err := s.client.SAdd(ctx, sessionsSetKey, id).Err(); err != nil {
		_ = s.client.Del(ctx, sessionKey(id)).Err()
		return "", contract.ErrInternal
	}
	return id, nil
}

func (s *redisStore) Open(ctx context.Context, id string) ([]contract.Message, error) {
	doc, err := s.loadDoc(ctx, id)
	if err != nil {
		return nil, err
	}
	messages := cloneMessages(doc.Messages)
	if messages == nil {
		messages = []contract.Message{}
	}
	return messages, nil
}

func (s *redisStore) TryBeginRun(ctx context.Context, id string) (RunSession, error) {
	exists, err := s.client.Exists(ctx, sessionKey(id)).Result()
	if err != nil {
		return nil, contract.ErrInternal
	}
	if exists == 0 {
		return nil, contract.ErrSessionNotFound
	}
	token := uuid.New().String()
	ok, err := s.client.SetNX(ctx, lockKey(id), token, s.lockTTL).Result()
	if err != nil {
		return nil, contract.ErrInternal
	}
	if !ok {
		return nil, contract.ErrSessionBusy
	}
	doc, err := s.loadDoc(ctx, id)
	if err != nil {
		_, _ = s.client.Eval(ctx, luaUnlock, []string{lockKey(id)}, token).Result()
		return nil, err
	}
	run := &redisRunSession{
		store: s,
		ctx:   ctx,
		id:    id,
		token: token,
		doc:   doc,
	}
	run.startRenew()
	return run, nil
}

func (s *redisStore) loadDoc(ctx context.Context, id string) (sessionDoc, error) {
	raw, err := s.client.Get(ctx, sessionKey(id)).Bytes()
	if err == redis.Nil {
		return sessionDoc{}, contract.ErrSessionNotFound
	}
	if err != nil {
		return sessionDoc{}, contract.ErrInternal
	}
	var doc sessionDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return sessionDoc{}, contract.ErrInternal
	}
	if doc.Messages == nil {
		doc.Messages = []contract.Message{}
	}
	return doc, nil
}

func (s *redisStore) persist(ctx context.Context, id, token string, doc sessionDoc) error {
	payload, err := json.Marshal(doc)
	if err != nil {
		return contract.ErrInternal
	}
	n, err := s.client.Eval(ctx, luaPersist, []string{lockKey(id), sessionKey(id)}, token, string(payload)).Int()
	if err != nil {
		return contract.ErrInternal
	}
	if n == 0 {
		return contract.ErrInternal
	}
	return nil
}

type redisRunSession struct {
	store       *redisStore
	ctx         context.Context
	id          string
	token       string
	doc         sessionDoc
	releaseOnce sync.Once
	stopRenew   chan struct{}
	renewDone   sync.WaitGroup
}

func (r *redisRunSession) startRenew() {
	if r.store.renewEvery <= 0 {
		return
	}
	r.stopRenew = make(chan struct{})
	r.renewDone.Add(1)
	go func() {
		defer r.renewDone.Done()
		ticker := time.NewTicker(r.store.renewEvery)
		defer ticker.Stop()
		for {
			select {
			case <-r.stopRenew:
				return
			case <-ticker.C:
				_ = r.store.client.Eval(
					context.Background(),
					luaRenew,
					[]string{lockKey(r.id)},
					r.token,
					r.store.lockTTL.Milliseconds(),
				).Err()
			}
		}
	}()
}

func (r *redisRunSession) ID() string { return r.id }

func (r *redisRunSession) Snapshot() []contract.Message {
	return cloneMessages(r.doc.Messages)
}

func (r *redisRunSession) Length() int { return len(r.doc.Messages) }

func (r *redisRunSession) UsedBytes() int { return r.doc.Bytes }

func (r *redisRunSession) Append(msgs ...contract.Message) error {
	added := 0
	for _, message := range msgs {
		added += messageBytes(message)
	}
	next := sessionDoc{
		Messages: append(cloneMessages(r.doc.Messages), cloneMessages(msgs)...),
		Bytes:    r.doc.Bytes + added,
	}
	if err := r.store.persist(r.ctx, r.id, r.token, next); err != nil {
		return err
	}
	r.doc = next
	return nil
}

func (r *redisRunSession) TruncateTo(n int) {
	if n < 0 || n >= len(r.doc.Messages) {
		return
	}
	next := sessionDoc{
		Messages: cloneMessages(r.doc.Messages[:n]),
		Bytes:    r.doc.Bytes,
	}
	for _, message := range r.doc.Messages[n:] {
		next.Bytes -= messageBytes(message)
	}
	if next.Bytes < 0 {
		next.Bytes = 0
	}
	if err := r.store.persist(r.ctx, r.id, r.token, next); err != nil {
		return
	}
	r.doc = next
}

func (r *redisRunSession) Release() {
	r.releaseOnce.Do(func() {
		if r.stopRenew != nil {
			close(r.stopRenew)
			r.renewDone.Wait()
		}
		_, _ = r.store.client.Eval(context.Background(), luaUnlock, []string{lockKey(r.id)}, r.token).Result()
	})
}
