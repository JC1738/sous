package logging

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Level is the "level" of a log message (e.g. debug vs fatal)
type Level int

//go:generate stringer -type Level
const (
	// CriticalLevel is the level for logging critical errors.
	CriticalLevel Level = iota

	// WarningLevel is the level for messages that may be problematic.
	WarningLevel

	// InformationLevel is for messages generated during normal operation.
	InformationLevel

	// DebugLevel is for messages primarily of interest to the software's developers.
	DebugLevel

	// ExtraDebug1Level is the first level of "super" debug messages.
	ExtraDebug1Level

	// ExtremeLevel is the "maximum" log level
	ExtremeLevel
)

func levelFromString(name string) Level {
	name = strings.ToLower(name)
	plusLevel := name + "level"

	for i := CriticalLevel; i <= ExtremeLevel; i++ {
		if strings.ToLower(i.String()) == name || strings.ToLower(i.String()) == plusLevel {
			return i
		}
	}

	fmt.Printf("Logging: invalid level %q; falling back to Extreme.", name)

	return ExtremeLevel
}

func (lvl Level) logrusLevel() logrus.Level {
	switch lvl {
	default:
		return logrus.DebugLevel
	case CriticalLevel:
		return logrus.ErrorLevel
	case WarningLevel:
		return logrus.WarnLevel
	case InformationLevel:
		return logrus.InfoLevel
	case DebugLevel:
		return logrus.DebugLevel
	case ExtraDebug1Level:
		return logrus.DebugLevel
	}
}
