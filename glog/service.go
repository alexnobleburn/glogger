package glog

import (
	"fmt"
	"github.com/alexnobleburn/glogger/glog/interfaces"
	"github.com/alexnobleburn/glogger/glog/models"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultInputBufferSize = 100
	defaultJobBufferSize   = 1000
	defaultNumWorkers      = 4
	defaultSendTimeout     = 100 * time.Millisecond
)

// defaultErrorHandler writes errors to stderr-style output.
var defaultErrorHandler = func(err error) {
	fmt.Println(err)
}

// ServiceOption configures LoggerService.
type ServiceOption func(*LoggerService)

func WithInputBufferSize(size int) ServiceOption {
	return func(ls *LoggerService) {
		if size > 0 {
			ls.inputBufferSize = size
		}
	}
}

func WithJobBufferSize(size int) ServiceOption {
	return func(ls *LoggerService) {
		if size > 0 {
			ls.jobBufferSize = size
		}
	}
}

func WithNumWorkers(n int) ServiceOption {
	return func(ls *LoggerService) {
		if n > 0 {
			ls.numWorkers = n
		}
	}
}

func WithSendTimeout(d time.Duration) ServiceOption {
	return func(ls *LoggerService) {
		if d > 0 {
			ls.sendTimeout = d
		}
	}
}

func WithErrorHandler(handler func(error)) ServiceOption {
	return func(ls *LoggerService) {
		if handler != nil {
			ls.errorHandler = handler
		}
	}
}

type LoggerService struct {
	inputCh        chan *models.LogData
	jobCh          chan sendJob
	inputBufferSize int
	jobBufferSize   int
	numWorkers     int
	sendTimeout    time.Duration
	errorHandler   func(error)
	mutex          sync.RWMutex
	loggers        map[string]interfaces.LogPublisher
	wg             sync.WaitGroup
	mainWg         sync.WaitGroup
	stopped        atomic.Bool
	stopOnce       sync.Once
}

func NewLoggerService(opts ...ServiceOption) *LoggerService {
	ls := &LoggerService{
		inputBufferSize: defaultInputBufferSize,
		jobBufferSize:   defaultJobBufferSize,
		loggers:         make(map[string]interfaces.LogPublisher),
		numWorkers:      defaultNumWorkers,
		sendTimeout:     defaultSendTimeout,
		errorHandler:    defaultErrorHandler,
	}
	for _, opt := range opts {
		opt(ls)
	}
	ls.inputCh = make(chan *models.LogData, ls.inputBufferSize)
	ls.jobCh = make(chan sendJob, ls.jobBufferSize)
	return ls
}

func (ls *LoggerService) AddLogger(loggerID string, logger interfaces.LogPublisher) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	ls.loggers[loggerID] = logger
}

func (ls *LoggerService) RemoveLogger(loggerID string) {
	ls.mutex.Lock()
	defer ls.mutex.Unlock()
	delete(ls.loggers, loggerID)
}

func (ls *LoggerService) GetInputChan() chan<- *models.LogData {
	return ls.inputCh
}

// NewLogger creates a Logger bound to this service.
func (ls *LoggerService) NewLogger() *Logger {
	return &Logger{
		logChan: ls.inputCh,
		stopped: &ls.stopped,
	}
}

func (ls *LoggerService) Start() {
	ls.mainWg.Add(1)
	go ls.runMainWorker()

	ls.wg.Add(ls.numWorkers)
	for i := 0; i < ls.numWorkers; i++ {
		go ls.runWorker()
	}
}

func (ls *LoggerService) Stop() {
	ls.stopOnce.Do(func() {
		ls.stopped.Store(true)
		close(ls.inputCh)
	})

	ls.mainWg.Wait()
	ls.wg.Wait()
}

func (ls *LoggerService) runMainWorker() {
	defer ls.mainWg.Done()
	defer close(ls.jobCh)
	for logData := range ls.inputCh {
		ls.processLogData(logData)
	}
}

func (ls *LoggerService) processLogData(logData *models.LogData) {
	if logData == nil {
		return
	}

	ls.mutex.RLock()
	if len(ls.loggers) == 0 {
		ls.mutex.RUnlock()
		ls.errorHandler(fmt.Errorf("glogger: no loggers configured, skipping log message"))
		return
	}

	jobs := make([]sendJob, 0, len(ls.loggers))
	for id, logger := range ls.loggers {
		if logger == nil {
			ls.errorHandler(fmt.Errorf("glogger: logger with ID %q is nil, skipping", id))
			continue
		}
		jobs = append(jobs, sendJob{
			loggerID: id,
			logger:   logger,
			logData:  logData,
		})
	}
	ls.mutex.RUnlock()

	for _, job := range jobs {
		ls.jobCh <- job
	}
}

func (ls *LoggerService) runWorker() {
	defer ls.wg.Done()
	for job := range ls.jobCh {
		ls.processJob(job)
	}
}

func (ls *LoggerService) processJob(job sendJob) {
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		defer func() {
			if r := recover(); r != nil {
				ls.errorHandler(fmt.Errorf("glogger: panic in publisher %q: %v", job.loggerID, r))
			}
		}()
		job.logger.SendMsg(job.logData)
	}()

	timer := time.NewTimer(ls.sendTimeout)
	defer timer.Stop()

	select {
	case <-doneCh:
	case <-timer.C:
		ls.errorHandler(fmt.Errorf(
			"glogger: timeout sending to publisher %q after %v, message: %q",
			job.loggerID, ls.sendTimeout, job.logData.Msg,
		))
	}
}

type sendJob struct {
	loggerID string
	logger   interfaces.LogPublisher
	logData  *models.LogData
}
