package storage

import (
	"time"

	"github.com/opentable/sous/util/logging"
)

type sqlMessage struct {
	logging.CallerInfo
	logging.MessageInterval
	sql      string
	rowcount int
	err      error
}

func newSQLMessage(started time.Time, sql string, rowcount int, err error) *sqlMessage {
	return &sqlMessage{
		CallerInfo:      logging.GetCallerInfo(logging.NotHere()),
		MessageInterval: logging.NewInterval(started, time.Now()),
		sql:             sql,
		rowcount:        rowcount,
		err:             err,
	}
}

func reportSQLMessage(log logging.LogSink, started time.Time, sql string, rowcount int, err error) {
	msg := newSQLMessage(started, sql, rowcount, err)
	msg.ExcludeMe()
	logging.Deliver(msg, log)
}

// DefaultLevel implements LogMessage on sqlMessage
func (msg *sqlMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
}

// Message implements LogMessage on sqlMessage
func (msg *sqlMessage) Message() string {
	if msg.err == nil {
		return "SQL query"
	}
	return msg.err.Error()
}

// EachField implements LogMessage on sqlMessage
func (msg *sqlMessage) EachField(fn logging.FieldReportFn) {
	msg.CallerInfo.EachField(fn)
	msg.MessageInterval.EachField(fn)
	fn("sous-sql-query", msg.sql)
	fn("sous-sql-rows", msg.rowcount)
	if msg.err != nil {
		fn("sous-sql-errreturned", msg.err.Error())
	}
}
