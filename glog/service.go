package glog

import (
	"fmt"
	"github.com/alexnobleburn/glogger/glog/interfaces"
	"github.com/alexnobleburn/glogger/glog/models"
	"sync"
	"time"
)

const (
	defaultInputBufferSize = 100
	defaultJobBufferSize   = 1000
	defaultNumWorkers      = 4
	sendTimeout            = 100 * time.Millisecond
)

type LoggerService struct {
	inputCh    chan *models.LogData
	jobCh      chan sendJob
	stopCh     chan struct{}
	numWorkers int
	mutex      sync.RWMutex
	loggers    map[string]interfaces.LogPublisher
}

func NewLoggerService(stopCh chan struct{}) *LoggerService {
	return &LoggerService{
		inputCh:    make(chan *models.LogData, defaultInputBufferSize),
		jobCh:      make(chan sendJob, defaultJobBufferSize),
		loggers:    make(map[string]interfaces.LogPublisher),
		stopCh:     stopCh,
		numWorkers: defaultNumWorkers,
	}
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

func (ls *LoggerService) Start() {
	go ls.runMainWorker()

	for i := 0; i < ls.numWorkers; i++ {
		go ls.runWorker()
	}
}

func (ls *LoggerService) runMainWorker() {
	defer close(ls.jobCh)
	for {
		select {
		case <-ls.stopCh:
			return
		case logData := <-ls.inputCh:
			if logData == nil {
				continue
			}
			ls.mutex.RLock()
			if len(ls.loggers) == 0 {
				fmt.Println("No loggers configured. Skipping log message.")
				continue
			}
			for id, logger := range ls.loggers {
				if logger == nil {
					fmt.Printf("Logger with ID %q is nil. Skipping.\n", id)
					continue
				}
				job := sendJob{
					loggerID: id,
					logger:   logger,
					logData:  logData,
				}

				ls.jobCh <- job
			}
			ls.mutex.RUnlock()
		}
	}
}

func (ls *LoggerService) runWorker() {
	for job := range ls.jobCh {
		ls.processJob(job)
	}
}

func (ls *LoggerService) processJob(job sendJob) {
	doneCh := make(chan struct{})
	go func() {
		job.logger.SendMsg(job.logData)
		close(doneCh)
	}()

	select {
	case <-doneCh:
	case <-time.After(sendTimeout):
		fmt.Printf(
			"Failed to send log message to logger %q within %v. Original message: %q\n",
			job.loggerID, sendTimeout, job.logData.Msg,
		)
	}
}

type sendJob struct {
	loggerID string
	logger   interfaces.LogPublisher
	logData  *models.LogData
}
