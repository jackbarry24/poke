// filepath: /Users/jb/dev/poke/test/echo_test.go
package test

import (
	"net/http"
	"testing"
	"time"

	"poke/core"
	"poke/types"
)

func TestPokeGet(t *testing.T) {
	go RunEchoServer(8083)
	time.Sleep(100 * time.Millisecond)

	runner := core.NewRequestRunner(nil)

	req := &types.PokeRequest{
		Method:  "GET",
		FullURL: "http://localhost:8083/echo",
	}

	res, err := runner.Send(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
}
