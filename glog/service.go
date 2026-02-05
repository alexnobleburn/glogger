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
	wg         sync.WaitGroup
	mainWg     sync.WaitGroup
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
	ls.mainWg.Add(1)
	go ls.runMainWorker()

	ls.wg.Add(ls.numWorkers)
	for i := 0; i < ls.numWorkers; i++ {
		go ls.runWorker()
	}
}

func (ls *LoggerService) Stop() {
	// Close input channel to signal no more logs will be sent
	close(ls.inputCh)

	// Wait for main worker to drain input channel and close job channel
	ls.mainWg.Wait()

	// Wait for all workers to finish processing jobs
	ls.wg.Wait()
}

func (ls *LoggerService) runMainWorker() {
	defer ls.mainWg.Done()
	defer close(ls.jobCh)
	for {
		select {
		case <-ls.stopCh:
			// Drain remaining logs from input channel
			for logData := range ls.inputCh {
				ls.processLogData(logData)
			}
			return
		case logData, ok := <-ls.inputCh:
			if !ok {
				return
			}
			ls.processLogData(logData)
		}
	}
}

func (ls *LoggerService) processLogData(logData *models.LogData) {
	if logData == nil {
		return
	}

	// Copy loggers while holding lock
	ls.mutex.RLock()
	if len(ls.loggers) == 0 {
		ls.mutex.RUnlock()
		fmt.Println("No loggers configured. Skipping log message.")
		return
	}

	// Create a slice of jobs while holding lock
	jobs := make([]sendJob, 0, len(ls.loggers))
	for id, logger := range ls.loggers {
		if logger == nil {
			fmt.Printf("Logger with ID %q is nil. Skipping.\n", id)
			continue
		}
		jobs = append(jobs, sendJob{
			loggerID: id,
			logger:   logger,
			logData:  logData,
		})
	}
	ls.mutex.RUnlock()

	// Send jobs without holding lock
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
