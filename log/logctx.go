package log

import (
	"github.com/apache/pulsar-client-go/pulsar/log"
)

type LogCtx struct {
}

func (l LogCtx) SubLogger(fields log.Fields) log.Logger {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) WithFields(fields log.Fields) log.Entry {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) WithField(name string, value interface{}) log.Entry {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) WithError(err error) log.Entry {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Debug(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Info(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Warn(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Error(args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Debugf(format string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Infof(format string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Warnf(format string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}

func (l LogCtx) Errorf(format string, args ...interface{}) {
	//TODO implement me
	panic("implement me")
}
