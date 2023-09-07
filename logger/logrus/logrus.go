package logrus

import (
	"github.com/sirupsen/logrus"
	"github.com/topfreegames/eventsgateway/v4/logger"
)

type logrusImpl struct {
	impl logrus.FieldLogger
}

// New returns a new logger.Logger implementation based on logrus
func New() logger.Logger {
	log := logrus.New()
	return NewWithLogger(log)
}

// NewWithLogger returns a new logger.Logger implementation based on a provided logrus instance
func NewWithLogger(logger logrus.FieldLogger) logger.Logger {
	return &logrusImpl{impl: logger}
}

func (l *logrusImpl) Fatal(format ...interface{}) {
	l.impl.Fatal(format...)
}

func (l *logrusImpl) Fatalf(format string, args ...interface{}) {
	l.impl.Fatalf(format, args...)
}

func (l *logrusImpl) Fatalln(args ...interface{}) {
	l.impl.Fatalln(args...)
}

func (l *logrusImpl) Debug(args ...interface{}) {
	l.impl.Debug(args...)
}

func (l *logrusImpl) Debugf(format string, args ...interface{}) {
	l.impl.Debugf(format, args...)
}

func (l *logrusImpl) Debugln(args ...interface{}) {
	l.impl.Debugln(args...)
}

func (l *logrusImpl) Error(args ...interface{}) {
	l.impl.Error(args...)
}

func (l *logrusImpl) Errorf(format string, args ...interface{}) {
	l.impl.Errorf(format, args...)
}

func (l *logrusImpl) Errorln(args ...interface{}) {
	l.impl.Errorln(args...)
}

func (l *logrusImpl) Info(args ...interface{}) {
	l.impl.Info(args...)
}

func (l *logrusImpl) Infof(format string, args ...interface{}) {
	l.impl.Infof(format, args...)
}

func (l *logrusImpl) Infoln(args ...interface{}) {
	l.impl.Infoln(args...)
}

func (l *logrusImpl) Warn(args ...interface{}) {
	l.impl.Warn(args...)
}

func (l *logrusImpl) Warnf(format string, args ...interface{}) {
	l.impl.Warnf(format, args...)
}

func (l *logrusImpl) Warnln(args ...interface{}) {
	l.impl.Warnln(args...)
}

func (l *logrusImpl) Panic(args ...interface{}) {
	l.impl.Panic(args...)
}

func (l *logrusImpl) Panicf(format string, args ...interface{}) {
	l.impl.Panicf(format, args...)
}

func (l *logrusImpl) Panicln(args ...interface{}) {
	l.impl.Panicln(args...)
}

func (l *logrusImpl) WithFields(fields map[string]interface{}) logger.Logger {
	return &logrusImpl{impl: l.impl.WithFields(fields)}

}
func (l *logrusImpl) WithField(key string, value interface{}) logger.Logger {
	return &logrusImpl{impl: l.impl.WithField(key, value)}
}

func (l *logrusImpl) WithError(err error) logger.Logger {
	return &logrusImpl{impl: l.impl.WithError(err)}
}
