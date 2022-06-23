package mysql_test

import (
	"testing"

	"github.com/tools-go/go-utils/mysql/instance"
)

func TestNewClient(t *testing.T) {
	cli := instance.GetMySQLClient()
	if cli == nil {
		t.Fatal("create mysql client failed")
	}

}
