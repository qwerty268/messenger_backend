package testing_boilerplate

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func MustLog(_ *testing.T) *logrus.Logger {
	l := logrus.New()
	return l
}
