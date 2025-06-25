package shopservice

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const requestNums = 100

var (
	successCount int64
	sendCount    int64
)

func TestSingleflight(t *testing.T) {
	// Reset counters before testing
	atomic.StoreInt64(&sendCount, 0)
	atomic.StoreInt64(&successCount, 0)

	var wg sync.WaitGroup
	wg.Add(requestNums)
	for range requestNums {
		go func() {
			defer wg.Done()
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get("http://localhost:14530/shop/7")
			defer func(Body io.ReadCloser) {
				_ = Body.Close()
			}(resp.Body)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}

			atomic.AddInt64(&sendCount, 1)
			if resp.StatusCode == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}

	wg.Wait()
	t.Logf("Total requests sent: %d, Successful responses: %d", sendCount, successCount)
	//t.Logf("Singleflight request count: %d", atomic.LoadInt64(&requestCount))
}
