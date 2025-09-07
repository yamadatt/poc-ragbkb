package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// レスポンス時間テストの設定
type ResponseTimeTest struct {
	MaxResponseTime time.Duration
	TargetEndpoint  string
	TestDuration    time.Duration
	SampleSize      int
}

// レスポンス時間統計
type ResponseTimeStats struct {
	Times        []time.Duration
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	TimeoutCount int
	ErrorCount   int
	mutex        sync.Mutex
}

func NewResponseTimeStats() *ResponseTimeStats {
	return &ResponseTimeStats{
		Times:   make([]time.Duration, 0),
		MinTime: time.Duration(0),
	}
}

func (rts *ResponseTimeStats) AddTime(duration time.Duration, isTimeout, isError bool) {
	rts.mutex.Lock()
	defer rts.mutex.Unlock()

	if isTimeout {
		rts.TimeoutCount++
		return
	}

	if isError {
		rts.ErrorCount++
		return
	}

	rts.Times = append(rts.Times, duration)
	rts.TotalTime += duration

	if rts.MinTime == 0 || duration < rts.MinTime {
		rts.MinTime = duration
	}
	if duration > rts.MaxTime {
		rts.MaxTime = duration
	}
}

func (rts *ResponseTimeStats) Average() time.Duration {
	if len(rts.Times) == 0 {
		return 0
	}
	return rts.TotalTime / time.Duration(len(rts.Times))
}

func (rts *ResponseTimeStats) Percentile(p float64) time.Duration {
	if len(rts.Times) == 0 {
		return 0
	}

	// 時間順にソート（簡単なバブルソート）
	times := make([]time.Duration, len(rts.Times))
	copy(times, rts.Times)

	for i := 0; i < len(times); i++ {
		for j := i + 1; j < len(times); j++ {
			if times[i] > times[j] {
				times[i], times[j] = times[j], times[i]
			}
		}
	}

	index := int(float64(len(times)) * p / 100.0)
	if index >= len(times) {
		index = len(times) - 1
	}
	return times[index]
}

func (rts *ResponseTimeStats) SuccessRate() float64 {
	total := len(rts.Times) + rts.TimeoutCount + rts.ErrorCount
	if total == 0 {
		return 0
	}
	return float64(len(rts.Times)) / float64(total) * 100.0
}

// 5秒以内レスポンステスト
func TestResponseTimeUnder5Seconds(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	tests := []struct {
		name           string
		endpoint       string
		method         string
		payload        map[string]interface{}
		maxTime        time.Duration
		sampleSize     int
		successRateMin float64
	}{
		{
			name:           "ヘルスチェック",
			endpoint:       "/api/health",
			method:         "GET",
			maxTime:        1 * time.Second,
			sampleSize:     50,
			successRateMin: 99.0,
		},
		{
			name:           "文書一覧",
			endpoint:       "/api/documents",
			method:         "GET",
			maxTime:        2 * time.Second,
			sampleSize:     30,
			successRateMin: 95.0,
		},
		{
			name:     "文書アップロード",
			endpoint: "/api/documents",
			method:   "POST",
			payload: map[string]interface{}{
				"fileName": "test.txt",
				"fileSize": 1024,
				"fileType": "txt",
			},
			maxTime:        3 * time.Second,
			sampleSize:     20,
			successRateMin: 95.0,
		},
		{
			name:     "RAGクエリ",
			endpoint: "/api/queries",
			method:   "POST",
			payload: map[string]interface{}{
				"question":  "AWS Bedrockについて教えてください",
				"sessionId": "perf_test_session",
			},
			maxTime:        5 * time.Second,
			sampleSize:     15,
			successRateMin: 90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewResponseTimeStats()
			client := &http.Client{Timeout: tt.maxTime + 2*time.Second}

			t.Logf("テスト開始: %s (%d リクエスト, 最大 %v)", tt.name, tt.sampleSize, tt.maxTime)

			// 複数回リクエストを実行
			for i := 0; i < tt.sampleSize; i++ {
				start := time.Now()

				var resp *http.Response
				var err error

				if tt.method == "GET" {
					resp, err = client.Get(system.server.URL + tt.endpoint)
				} else if tt.method == "POST" {
					var body []byte
					if tt.payload != nil {
						body, err = json.Marshal(tt.payload)
						if err != nil {
							stats.AddTime(0, false, true)
							continue
						}
					}
					resp, err = client.Post(system.server.URL+tt.endpoint, "application/json", bytes.NewBuffer(body))
				}

				duration := time.Since(start)

				if err != nil {
					stats.AddTime(duration, duration > tt.maxTime, true)
					continue
				}

				resp.Body.Close()

				if resp.StatusCode >= 400 {
					stats.AddTime(duration, false, true)
					continue
				}

				isTimeout := duration > tt.maxTime
				stats.AddTime(duration, isTimeout, false)

				// リクエスト間に小さな間隔
				if i < tt.sampleSize-1 {
					time.Sleep(100 * time.Millisecond)
				}
			}

			// 結果の出力
			t.Logf("結果:")
			t.Logf("  成功リクエスト: %d/%d", len(stats.Times), tt.sampleSize)
			t.Logf("  タイムアウト: %d", stats.TimeoutCount)
			t.Logf("  エラー: %d", stats.ErrorCount)
			t.Logf("  成功率: %.2f%%", stats.SuccessRate())
			t.Logf("  平均時間: %v", stats.Average())
			t.Logf("  最小時間: %v", stats.MinTime)
			t.Logf("  最大時間: %v", stats.MaxTime)
			t.Logf("  50パーセンタイル: %v", stats.Percentile(50))
			t.Logf("  90パーセンタイル: %v", stats.Percentile(90))
			t.Logf("  95パーセンタイル: %v", stats.Percentile(95))
			t.Logf("  99パーセンタイル: %v", stats.Percentile(99))

			// アサーション
			assert.True(t, stats.SuccessRate() >= tt.successRateMin,
				fmt.Sprintf("成功率 %.2f%% が期待値 %.2f%% を下回っています", stats.SuccessRate(), tt.successRateMin))

			if len(stats.Times) > 0 {
				assert.True(t, stats.Percentile(95) <= tt.maxTime,
					fmt.Sprintf("95パーセンタイル %v が制限時間 %v を超えています", stats.Percentile(95), tt.maxTime))
				assert.True(t, stats.Average() <= tt.maxTime,
					fmt.Sprintf("平均応答時間 %v が制限時間 %v を超えています", stats.Average(), tt.maxTime))
			}
		})
	}
}

// 段階的負荷テスト
func TestGradualLoadIncrease(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	// 段階的に負荷を増加させる
	loadLevels := []struct {
		concurrency int
		duration    time.Duration
		name        string
	}{
		{1, 10 * time.Second, "軽負荷"},
		{3, 15 * time.Second, "中負荷"},
		{5, 10 * time.Second, "高負荷"},
	}

	for _, level := range loadLevels {
		t.Run(level.name, func(t *testing.T) {
			stats := NewResponseTimeStats()
			var wg sync.WaitGroup

			ctx, cancel := context.WithTimeout(context.Background(), level.duration)
			defer cancel()

			t.Logf("負荷レベル: %s (同時接続数: %d, 継続時間: %v)", level.name, level.concurrency, level.duration)

			// 並行ワーカー起動
			for i := 0; i < level.concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					requestCount := 0
					for {
						select {
						case <-ctx.Done():
							t.Logf("Worker %d completed %d requests", workerID, requestCount)
							return
						default:
							start := time.Now()

							resp, err := client.Get(system.server.URL + "/api/health")
							duration := time.Since(start)

							if err != nil {
								stats.AddTime(duration, false, true)
							} else {
								resp.Body.Close()
								isTimeout := duration > 5*time.Second
								isError := resp.StatusCode >= 400
								stats.AddTime(duration, isTimeout, isError)
							}

							requestCount++
							time.Sleep(time.Duration(500+workerID*100) * time.Millisecond)
						}
					}
				}(i)
			}

			wg.Wait()

			// 結果評価
			t.Logf("負荷テスト結果 (%s):", level.name)
			t.Logf("  総リクエスト: %d", len(stats.Times)+stats.ErrorCount+stats.TimeoutCount)
			t.Logf("  成功率: %.2f%%", stats.SuccessRate())
			t.Logf("  平均応答時間: %v", stats.Average())
			t.Logf("  95パーセンタイル: %v", stats.Percentile(95))

			// 各負荷レベルでの性能基準
			switch level.concurrency {
			case 1:
				assert.True(t, stats.SuccessRate() >= 99.0, "軽負荷時の成功率")
				assert.True(t, stats.Percentile(95) <= 1*time.Second, "軽負荷時の95パーセンタイル")
			case 3:
				assert.True(t, stats.SuccessRate() >= 95.0, "中負荷時の成功率")
				assert.True(t, stats.Percentile(95) <= 3*time.Second, "中負荷時の95パーセンタイル")
			case 5:
				assert.True(t, stats.SuccessRate() >= 90.0, "高負荷時の成功率")
				assert.True(t, stats.Percentile(95) <= 5*time.Second, "高負荷時の95パーセンタイル")
			}
		})
	}
}

// エンドポイント別パフォーマンス比較
func TestEndpointPerformanceComparison(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	endpoints := []struct {
		name     string
		path     string
		method   string
		payload  map[string]interface{}
		expected time.Duration
	}{
		{
			name:     "ヘルスチェック",
			path:     "/api/health",
			method:   "GET",
			expected: 100 * time.Millisecond,
		},
		{
			name:     "文書一覧",
			path:     "/api/documents",
			method:   "GET",
			expected: 200 * time.Millisecond,
		},
		{
			name:   "文書アップロード",
			path:   "/api/documents",
			method: "POST",
			payload: map[string]interface{}{
				"fileName": "perf_test.txt",
				"fileSize": 1024,
				"fileType": "txt",
			},
			expected: 500 * time.Millisecond,
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	results := make(map[string]*ResponseTimeStats)

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			stats := NewResponseTimeStats()
			results[endpoint.name] = stats

			// 10回テスト実行
			for i := 0; i < 10; i++ {
				start := time.Now()

				var resp *http.Response
				var err error

				if endpoint.method == "GET" {
					resp, err = client.Get(system.server.URL + endpoint.path)
				} else if endpoint.method == "POST" {
					var body []byte
					if endpoint.payload != nil {
						body, _ = json.Marshal(endpoint.payload)
					}
					resp, err = client.Post(system.server.URL+endpoint.path, "application/json", bytes.NewBuffer(body))
				}

				duration := time.Since(start)

				if err != nil {
					stats.AddTime(duration, false, true)
				} else {
					resp.Body.Close()
					isError := resp.StatusCode >= 400
					stats.AddTime(duration, false, isError)
				}

				time.Sleep(200 * time.Millisecond)
			}

			// 期待値との比較
			avgTime := stats.Average()
			t.Logf("%s - 平均応答時間: %v (期待値: %v)", endpoint.name, avgTime, endpoint.expected)

			// 期待値の2倍以内であることを確認（ゆるめの基準）
			assert.True(t, avgTime <= endpoint.expected*2,
				fmt.Sprintf("%s の応答時間 %v が期待値 %v の2倍を超えています", endpoint.name, avgTime, endpoint.expected))
		})
	}

	// 全体の比較結果
	t.Logf("\n=== エンドポイント性能比較 ===")
	for name, stats := range results {
		if len(stats.Times) > 0 {
			t.Logf("%-15s: 平均 %7v, 最小 %7v, 最大 %7v, 成功率 %5.1f%%",
				name, stats.Average(), stats.MinTime, stats.MaxTime, stats.SuccessRate())
		}
	}
}

// タイムアウト処理テスト
func TestTimeoutHandling(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	// 短いタイムアウト設定のクライアント
	client := &http.Client{Timeout: 100 * time.Millisecond}

	timeoutCount := 0
	successCount := 0

	// RAGクエリ（時間がかかる処理）に短いタイムアウトで送信
	for i := 0; i < 10; i++ {
		queryData := map[string]interface{}{
			"question":  "This query will likely timeout",
			"sessionId": fmt.Sprintf("timeout_test_%d", i),
		}

		jsonData, err := json.Marshal(queryData)
		require.NoError(t, err)

		start := time.Now()
		resp, err := client.Post(system.server.URL+"/api/queries", "application/json", bytes.NewBuffer(jsonData))
		duration := time.Since(start)

		if err != nil {
			t.Logf("Request %d: タイムアウト (%v)", i, duration)
			timeoutCount++
		} else {
			resp.Body.Close()
			t.Logf("Request %d: 成功 (%v)", i, duration)
			successCount++
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Logf("タイムアウトテスト結果:")
	t.Logf("  タイムアウト: %d", timeoutCount)
	t.Logf("  成功: %d", successCount)

	// タイムアウトが適切に発生していることを確認
	assert.True(t, timeoutCount > 0, "短いタイムアウト設定でタイムアウトが発生すること")
	assert.True(t, timeoutCount >= 5, "予想通りタイムアウトが多発すること")
}

// 長時間処理の性能監視
func TestLongRunningProcessPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("短時間テスト時はスキップ")
	}

	system := setupTestServer()
	defer system.server.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	stats := NewResponseTimeStats()

	// 5分間の継続テスト
	duration := 5 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	requestCount := 0

	t.Logf("長時間性能テスト開始 (継続時間: %v)", duration)

	for {
		select {
		case <-ctx.Done():
			t.Logf("長時間テスト完了: %d リクエスト処理", requestCount)
			goto TestComplete
		default:
			start := time.Now()

			resp, err := client.Get(system.server.URL + "/api/health")
			elapsed := time.Since(start)

			if err != nil {
				stats.AddTime(elapsed, false, true)
			} else {
				resp.Body.Close()
				isError := resp.StatusCode >= 400
				stats.AddTime(elapsed, false, isError)
			}

			requestCount++

			if requestCount%100 == 0 {
				t.Logf("Progress: %d requests, 平均応答時間: %v, 成功率: %.1f%%",
					requestCount, stats.Average(), stats.SuccessRate())
			}

			time.Sleep(1 * time.Second)
		}
	}

TestComplete:

	// 結果評価
	t.Logf("長時間性能テスト結果:")
	t.Logf("  総リクエスト数: %d", requestCount)
	t.Logf("  成功率: %.2f%%", stats.SuccessRate())
	t.Logf("  平均応答時間: %v", stats.Average())
	t.Logf("  95パーセンタイル: %v", stats.Percentile(95))
	t.Logf("  スループット: %.2f req/min", float64(len(stats.Times))/(duration.Minutes()))

	// 長時間実行での性能劣化がないことを確認
	assert.True(t, stats.SuccessRate() >= 95.0, "長時間実行での成功率維持")
	assert.True(t, stats.Average() <= 2*time.Second, "長時間実行での平均応答時間維持")
	assert.True(t, stats.Percentile(95) <= 5*time.Second, "長時間実行での95パーセンタイル維持")

	// 最低限のスループット確保
	throughput := float64(len(stats.Times)) / duration.Minutes()
	assert.True(t, throughput >= 30.0, "最低30req/minのスループット確保")
}

// ベンチマークテスト - レスポンス時間測定
func BenchmarkResponseTimes(b *testing.B) {
	system := setupTestServer()
	defer system.server.Close()

	client := &http.Client{Timeout: 10 * time.Second}

	benchmarks := []struct {
		name     string
		endpoint string
		method   string
	}{
		{"Health", "/api/health", "GET"},
		{"Documents", "/api/documents", "GET"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var resp *http.Response
				var err error

				if bm.method == "GET" {
					resp, err = client.Get(system.server.URL + bm.endpoint)
				}

				if err != nil {
					b.Error(err)
					continue
				}

				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					b.Errorf("Expected 200, got %d", resp.StatusCode)
				}
			}
		})
	}
}
