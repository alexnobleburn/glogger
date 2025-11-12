package interfaces

import (
	"github.com/alexnobleburn/glogger/glog/models"
)

type LogPublisher interface {
	SendMsg(data *models.LogData)
}
