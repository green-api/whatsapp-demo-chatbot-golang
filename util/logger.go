package util

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Formatter struct {
	Location  *time.Location
	Formatter *logrus.JSONFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Time = entry.Time.In(f.Location)

	return f.Formatter.Format(entry)
}
