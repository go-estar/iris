package iris

import (
	"github.com/google/uuid"
	"testing"
)

func TestNewBaseError(t *testing.T) {
	requestId := uuid.New().String()
	t.Log(requestId)
}
