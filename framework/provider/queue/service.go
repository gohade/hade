package queue

import (
	"bytes"
	"context"
	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/provider/queue/engine"
	"github.com/gohade/hade/framework/util/goroutine"
	"github.com/pkg/errors"
	"strconv"
	"sync"
)

type HadeQueueService struct {
	container framework.Container
	lock      sync.Locker

	registerJobs   map[string]contract.Job
	connectionJobs map[string][]contract.Job
	engines        map[string]engine.QueueEngine
}

func (h *HadeQueueService) Register(job contract.Job) {
	h.lock.Lock()
	defer h.lock.Unlock()

	jobName := job.JobName()
	// 计算connName
	configPath := h.getJobConnection(job)

	// 最终的注册结果
	registerFail := false

	defer func() {
		findIndex := func(jobs []contract.Job, job contract.Job) int {
			for i, t := range jobs {
				if t.JobName() == job.JobName() {
					return i
				}
			}
			return -1
		}
		if registerFail == true {
			if jobs, ok := h.connectionJobs[configPath]; ok {
				if index := findIndex(jobs, job); index != -1 {
					if len(jobs) == 1 {
						jobs = jobs[0:0]
					} else {
						jobs = append(jobs[:index], jobs[index+1:]...) // 删除中间1个元素
					}
				}
				h.connectionJobs[configPath] = jobs
			}
			if len(h.connectionJobs[configPath]) == 0 {
				// delete engine
				if _, ok := h.engines[configPath]; ok {
					delete(h.engines, configPath)
				}
			}

			delete(h.registerJobs, jobName)
		}
	}()

	// 填充registerJobs
	h.registerJobs[jobName] = job

	// 填充connectionJobs
	if h.connectionJobs == nil {
		h.connectionJobs = make(map[string][]contract.Job)
	}
	if _, ok := h.connectionJobs[configPath]; ok {
		h.connectionJobs[configPath] = make([]contract.Job, 0)
	}
	h.connectionJobs[configPath] = append(h.connectionJobs[configPath], job)

	// 填充engines
	logger := h.container.MustMake(contract.LogKey).(contract.Log)
	if qEngine, ok := h.engines[configPath]; !ok || qEngine == nil {
		queueEngine, err := engine.NewQueueEngine(h.container, configPath)
		if err != nil {
			logger.Error(context.Background(), "engine.NewQueueEngine error", map[string]interface{}{
				"error": err,
				"path":  configPath,
			})
			registerFail = true
			return
		}
		h.engines[configPath] = queueEngine
	}
	return
}

func (h *HadeQueueService) Listen() error {
	h.lock.Lock()
	defer h.lock.Unlock()

	configService := h.container.MustMake(contract.ConfigKey).(contract.Config)
	logger := h.container.MustMake(contract.LogKey).(contract.Log)

	for key, queueEngine := range h.engines {
		signal := make(chan int)
		goroutine.SafeGo(context.Background(), func() {
			pop, err := queueEngine.Listen(signal)
			if err != nil {
				logger.Error(context.Background(), "listen engine error", map[string]interface{}{
					"error":     err,
					"conn_path": key,
				})
				return
			}

			var elem []byte
			for {
				select {
				case elem = <-pop:
					jobName, body, err := h.parsePopBytes(elem)
					if err != nil {
						logger.Error(context.Background(), "listener pop element error", map[string]interface{}{
							"error":     err,
							"conn_path": key,
							"elem":      elem,
						})
						continue
					}
					job := h.findJobByJobName(jobName)
					if job == nil {
						logger.Error(context.Background(), "listener element job can not find register job", map[string]interface{}{
							"error":     err,
							"conn_path": key,
							"job_name":  jobName,
						})
						continue
					}
					if err := job.UnmarshalText(body); err != nil {
                        logger.Error(context.Background(), "listener element job unmarshal error", map[string]interface{}{
                            "error":     err,
                            "conn_path": key,
                            "job_name":  jobName,
                        })
                        continue
                    }
                    job.Fire(context.Background(),)
				}
			}
		})
	}
}

func (h *HadeQueueService) jobHeader(job contract.Job) []byte {
	buf := new(bytes.Buffer)

	jobName := job.JobName()
	jobNameBytes := []byte(jobName)
	length := len(jobNameBytes)
	lengthByte := []byte(strconv.Itoa(length))
	if len(lengthByte) < 4 {
		for i := 1; i < 4-len(lengthByte); i++ {
			buf.WriteByte('0')
		}
	}
	buf.Write(lengthByte)
	buf.Write(jobNameBytes)
	return buf.Bytes()
}

func (h *HadeQueueService) parsePopBytes(content []byte) (string, []byte, error) {
	lengthByte := content[0:4]
	length, err := strconv.ParseInt(string(lengthByte), 10, 64)
	if err != nil {
		return "", nil, err
	}
	if len(content) < int(length)+4 {
		return "", nil, errors.New("pop length error, can not parse job name")
	}
	jobNameBytes := content[4 : length+4]
	return string(jobNameBytes), content[length+4:], nil
}

func (h *HadeQueueService) Push(ctx context.Context, job contract.Job) error {
	text, err := job.MarshalText()
	if err != nil {
		return err
	}
	jobHeader := h.jobHeader(job)
	jobHeader = append(jobHeader, text...)
	queueEngine := h.findQueueEngine(job)
	if queueEngine == nil {
		return errors.New("queue engine is empty: " + job.JobName())
	}
	return queueEngine.Push(jobHeader)
}

func (h *HadeQueueService) GoPush(ctx context.Context, job contract.Job) {
	goroutine.SafeGo(ctx, func() {
		h.Push(ctx, job)
	})
}

func NewHadeQueueService(params ...interface{}) (interface{}, error) {
	c := params[0].(framework.Container)
	lock := &sync.RWMutex{}
	return &HadeQueueService{container: c, lock: lock}, nil
}

func (h *HadeQueueService) getJobConnection(job contract.Job) string {
	// 计算connName
	configPath := "queue.default"
	if conn, ok := job.(contract.OnConnection); ok {
		configPath = conn.OnConnection()
	}
	return configPath
}

func (h *HadeQueueService) findConnection(job contract.Job) string {
	for connection, jobs := range h.connectionJobs {
		for _, t := range jobs {
			if t.JobName() == job.JobName() {
				return connection
			}
		}
	}
	return ""
}

func (h *HadeQueueService) findQueueEngine(job contract.Job) engine.QueueEngine {
	connection := h.findConnection(job)
	if connection == "" {
		return nil
	}
	if queueEngine, ok := h.engines[connection]; ok && queueEngine != nil {
		return queueEngine
	}
	return nil
}

func (h *HadeQueueService) findJobByJobName(jobName string) contract.Job {
	if job, ok := h.registerJobs[jobName]; ok && job != nil {
		return job
	}
	return nil
}
