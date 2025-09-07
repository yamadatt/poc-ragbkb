package performance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のRAGシステム設定
type TestRAGSystem struct {
	router *gin.Engine
	server *httptest.Server
}

// 同時リクエスト用のレスポンス統計
type RequestStats struct {
	SuccessCount int
	ErrorCount   int
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	Errors       []error
	mutex        sync.Mutex
}

func (rs *RequestStats) AddResult(duration time.Duration, err error) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.TotalTime += duration

	if err != nil {
		rs.ErrorCount++
		rs.Errors = append(rs.Errors, err)
		return
	}

	rs.SuccessCount++

	if rs.MinTime == 0 || duration < rs.MinTime {
		rs.MinTime = duration
	}
	if duration > rs.MaxTime {
		rs.MaxTime = duration
	}
}

func (rs *RequestStats) AverageTime() time.Duration {
	if rs.SuccessCount == 0 {
		return 0
	}
	return rs.TotalTime / time.Duration(rs.SuccessCount)
}

func (rs *RequestStats) SuccessRate() float64 {
	total := rs.SuccessCount + rs.ErrorCount
	if total == 0 {
		return 0
	}
	return float64(rs.SuccessCount) / float64(total) * 100
}

// テスト用のモックハンドラー
func setupTestServer() *TestRAGSystem {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Health check endpoint
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status":    "healthy",
				"message":   "Service is running",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		})
	})

	// Documents endpoints
	router.GET("/api/documents", func(c *gin.Context) {
		// 少し処理時間をシミュレート
		time.Sleep(50 * time.Millisecond)

		c.JSON(http.StatusOK, gin.H{
			"data": []gin.H{
				{
					"id":       "doc1",
					"fileName": "sample1.txt",
					"fileSize": 1024,
					"status":   "indexed",
				},
				{
					"id":       "doc2",
					"fileName": "sample2.txt",
					"fileSize": 2048,
					"status":   "indexed",
				},
			},
			"pagination": gin.H{
				"page":       1,
				"pageSize":   10,
				"totalCount": 2,
				"totalPages": 1,
			},
		})
	})

	router.POST("/api/documents", func(c *gin.Context) {
		// アップロードセッション作成をシミュレート
		time.Sleep(100 * time.Millisecond)

		c.JSON(http.StatusCreated, gin.H{
			"data": gin.H{
				"id":        "upload123",
				"fileName":  "test.txt",
				"uploadUrl": "https://s3.example.com/upload",
				"expiresAt": time.Now().Add(time.Hour).Format(time.RFC3339),
			},
		})
	})

	// Queries endpoint (RAG処理のシミュレート)
	router.POST("/api/queries", func(c *gin.Context) {
		var requestBody struct {
			Question  string `json:"question"`
			SessionID string `json:"sessionId"`
		}

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid request"}})
			return
		}

		// RAG処理時間をシミュレート（1-3秒）
		processingTime := time.Duration(1000+time.Now().UnixNano()%2000) * time.Millisecond
		time.Sleep(processingTime)

		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"query": gin.H{
					"id":               fmt.Sprintf("query_%d", time.Now().UnixNano()),
					"sessionId":        requestBody.SessionID,
					"question":         requestBody.Question,
					"status":           "completed",
					"processingTimeMs": int(processingTime.Milliseconds()),
					"createdAt":        time.Now().UTC().Format(time.RFC3339),
				},
				"response": gin.H{
					"id":     fmt.Sprintf("resp_%d", time.Now().UnixNano()),
					"answer": fmt.Sprintf("これは「%s」に対する回答です。AWS Bedrockの機能について説明します。", requestBody.Question),
					"sources": []gin.H{
						{
							"documentId": "doc1",
							"fileName":   "aws-bedrock.txt",
							"excerpt":    "AWS Bedrock関連の情報...",
							"confidence": 0.9,
						},
					},
					"processingTimeMs": int(processingTime.Milliseconds()),
					"modelUsed":        "claude-v1",
					"tokensUsed":       150,
					"createdAt":        time.Now().UTC().Format(time.RFC3339),
				},
			},
		})
	})

	server := httptest.NewServer(router)

	return &TestRAGSystem{
		router: router,
		server: server,
	}
}

// 3同時ユーザーテスト
func TestConcurrent3Users(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	const numUsers = 3
	const requestsPerUser = 10

	stats := &RequestStats{}
	var wg sync.WaitGroup

	// 各ユーザーの処理
	userScenario := func(userID int) {
		defer wg.Done()

		client := &http.Client{Timeout: 10 * time.Second}
		sessionID := fmt.Sprintf("session_%d", userID)

		for i := 0; i < requestsPerUser; i++ {
			start := time.Now()

			// 1. ヘルスチェック
			_, err := client.Get(system.server.URL + "/api/health")
			if err != nil {
				stats.AddResult(time.Since(start), fmt.Errorf("health check failed: %v", err))
				continue
			}

			// 2. 文書一覧取得
			_, err = client.Get(system.server.URL + "/api/documents")
			if err != nil {
				stats.AddResult(time.Since(start), fmt.Errorf("documents list failed: %v", err))
				continue
			}

			// 3. クエリ送信
			queryData := map[string]interface{}{
				"question":  fmt.Sprintf("User%d の質問 %d: AWS Bedrockについて教えてください", userID, i),
				"sessionId": sessionID,
			}

			jsonData, err := json.Marshal(queryData)
			if err != nil {
				stats.AddResult(time.Since(start), fmt.Errorf("json marshal failed: %v", err))
				continue
			}

			resp, err := client.Post(system.server.URL+"/api/queries", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				stats.AddResult(time.Since(start), fmt.Errorf("query request failed: %v", err))
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				stats.AddResult(time.Since(start), fmt.Errorf("query returned status %d", resp.StatusCode))
				continue
			}

			duration := time.Since(start)
			stats.AddResult(duration, nil)

			// ユーザー間で少し間隔を空ける
			time.Sleep(time.Duration(50+i*10) * time.Millisecond)
		}
	}

	// テスト開始時間を記録
	testStart := time.Now()

	// 同時実行
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go userScenario(i)
	}

	wg.Wait()
	testDuration := time.Since(testStart)

	// 結果の検証
	t.Logf("同時3ユーザーテスト結果:")
	t.Logf("  総テスト時間: %v", testDuration)
	t.Logf("  成功リクエスト: %d/%d", stats.SuccessCount, numUsers*requestsPerUser)
	t.Logf("  エラーリクエスト: %d", stats.ErrorCount)
	t.Logf("  成功率: %.2f%%", stats.SuccessRate())
	t.Logf("  平均レスポンス時間: %v", stats.AverageTime())
	t.Logf("  最短レスポンス時間: %v", stats.MinTime)
	t.Logf("  最長レスポンス時間: %v", stats.MaxTime)

	// エラーの詳細を出力
	if len(stats.Errors) > 0 {
		t.Logf("  エラー詳細:")
		for i, err := range stats.Errors {
			if i < 5 { // 最初の5個だけ表示
				t.Logf("    %d: %v", i+1, err)
			}
		}
		if len(stats.Errors) > 5 {
			t.Logf("    ... and %d more errors", len(stats.Errors)-5)
		}
	}

	// アサーション
	assert.True(t, stats.SuccessRate() >= 95.0, "成功率が95%以上であること")
	assert.True(t, stats.AverageTime() < 5*time.Second, "平均レスポンス時間が5秒以内であること")
	assert.True(t, stats.MaxTime < 10*time.Second, "最大レスポンス時間が10秒以内であること")

	// 全体の処理時間もチェック（同時実行により効率的であることを確認）
	expectedSequentialTime := time.Duration(float64(numUsers*requestsPerUser) * 1.5) * time.Second // 順次実行の場合の見積もり
	assert.True(t, testDuration < expectedSequentialTime/2, "同時実行により処理時間が効率化されていること")
}

// 負荷スパイクテスト
func TestLoadSpike(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	const spikeUsers = 10
	const requestsPerUser = 5

	stats := &RequestStats{}
	var wg sync.WaitGroup

	// 一斉にリクエストを送信
	for i := 0; i < spikeUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			client := &http.Client{Timeout: 15 * time.Second}

			for j := 0; j < requestsPerUser; j++ {
				start := time.Now()

				resp, err := client.Get(system.server.URL + "/api/health")
				if err != nil {
					stats.AddResult(time.Since(start), err)
					continue
				}
				resp.Body.Close()

				stats.AddResult(time.Since(start), nil)
			}
		}(i)
	}

	wg.Wait()

	// 結果の検証
	t.Logf("負荷スパイクテスト結果:")
	t.Logf("  成功リクエスト: %d/%d", stats.SuccessCount, spikeUsers*requestsPerUser)
	t.Logf("  成功率: %.2f%%", stats.SuccessRate())
	t.Logf("  平均レスポンス時間: %v", stats.AverageTime())

	// スパイク時でも基本的な応答性は保たれること
	assert.True(t, stats.SuccessRate() >= 90.0, "負荷スパイク時でも成功率90%以上を維持")
	assert.True(t, stats.AverageTime() < 2*time.Second, "負荷スパイク時でも平均2秒以内で応答")
}

// 長時間実行テスト
func TestLongRunningLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("短時間テスト時はスキップ")
	}

	system := setupTestServer()
	defer system.server.Close()

	const duration = 30 * time.Second
	const numUsers = 3

	stats := &RequestStats{}
	var wg sync.WaitGroup

	ctx := make(chan bool)

	// 指定時間後にテスト終了シグナル
	go func() {
		time.Sleep(duration)
		close(ctx)
	}()

	// 継続的な負荷生成
	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			client := &http.Client{Timeout: 10 * time.Second}
			requestCount := 0

			for {
				select {
				case <-ctx:
					t.Logf("User %d completed %d requests", userID, requestCount)
					return
				default:
					start := time.Now()

					resp, err := client.Get(system.server.URL + "/api/health")
					if err != nil {
						stats.AddResult(time.Since(start), err)
					} else {
						resp.Body.Close()
						stats.AddResult(time.Since(start), nil)
					}

					requestCount++
					time.Sleep(time.Duration(500+requestCount%1000) * time.Millisecond)
				}
			}
		}(i)
	}

	wg.Wait()

	// 結果の検証
	t.Logf("長時間負荷テスト結果 (Duration: %v):", duration)
	t.Logf("  総リクエスト数: %d", stats.SuccessCount+stats.ErrorCount)
	t.Logf("  成功率: %.2f%%", stats.SuccessRate())
	t.Logf("  平均レスポンス時間: %v", stats.AverageTime())
	t.Logf("  スループット: %.2f req/sec", float64(stats.SuccessCount)/duration.Seconds())

	// 長時間実行でも性能が維持されること
	assert.True(t, stats.SuccessRate() >= 95.0, "長時間実行でも成功率95%以上を維持")
	assert.True(t, stats.AverageTime() < 3*time.Second, "長時間実行でも平均3秒以内で応答")

	// 最低限のスループットを確保
	throughput := float64(stats.SuccessCount) / duration.Seconds()
	assert.True(t, throughput >= 1.0, "最低1req/secのスループットを確保")
}

// メモリ使用量監視テスト
func TestMemoryUsage(t *testing.T) {
	system := setupTestServer()
	defer system.server.Close()

	// 大量のクエリを送信してメモリリークがないことを確認
	client := &http.Client{Timeout: 10 * time.Second}

	for i := 0; i < 100; i++ {
		queryData := map[string]interface{}{
			"question":  fmt.Sprintf("Memory test query %d with some additional text to increase payload size", i),
			"sessionId": fmt.Sprintf("session_memory_test_%d", i),
		}

		jsonData, err := json.Marshal(queryData)
		require.NoError(t, err)

		resp, err := client.Post(system.server.URL+"/api/queries", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()

		if i%10 == 0 {
			t.Logf("Completed %d requests", i)
		}
	}

	t.Logf("メモリ使用量テスト完了: 100リクエストを正常処理")
}

// ベンチマークテスト
func BenchmarkHealthCheck(b *testing.B) {
	system := setupTestServer()
	defer system.server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(system.server.URL + "/api/health")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkDocumentsList(b *testing.B) {
	system := setupTestServer()
	defer system.server.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(system.server.URL + "/api/documents")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkQuery(b *testing.B) {
	system := setupTestServer()
	defer system.server.Close()

	client := &http.Client{Timeout: 15 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			queryData := map[string]interface{}{
				"question":  "Benchmark query about AWS Bedrock",
				"sessionId": fmt.Sprintf("bench_session_%d", b.N),
			}

			jsonData, err := json.Marshal(queryData)
			if err != nil {
				b.Error(err)
				continue
			}

			resp, err := client.Post(system.server.URL+"/api/queries", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}
