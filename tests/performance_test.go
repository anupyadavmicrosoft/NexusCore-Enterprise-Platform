package tests

import (
	"encoding/json"
	"testing"
)

// ==============================================================================
// PERFORMANCE BENCHMARK SUITE - Memory allocations and CPU Profiling
// Evaluates serialization efficiency (B/op, allocs/op) under concurrent pressure
// ==============================================================================

type BenchmarkData struct {
	TxID      string  `json:"tx_id"`
	AccountID string  `json:"account_id"`
	Amount    float64 `json:"amount"`
	Timestamp int64   `json:"timestamp"`
	IsAudit   bool    `json:"is_audit"`
}

// 1. Benchmark standard JSON marshalling (evaluate memory pressure and allocations)
func Benchmark_JSON_Serialization(b *testing.B) {
	data := BenchmarkData{
		TxID:      "tx-bench-0092123",
		AccountID: "acc-user-enterprise-9999",
		Amount:    829910.45,
		Timestamp: 1789123456,
		IsAudit:   true,
	}

	b.ResetTimer() // Exclude initialization cycles from benchmark results

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(&data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 2. Benchmark JSON Deserialization performance
func Benchmark_JSON_Deserialization(b *testing.B) {
	rawPayload := []byte(`{"tx_id":"tx-bench-0092123","account_id":"acc-user-enterprise-9999","amount":829910.45,"timestamp":1789123456,"is_audit":true}`)
	
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var dest BenchmarkData
		err := json.Unmarshal(rawPayload, &dest)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// 3. Parallel serialization evaluation representing API Gateway ingress under load
func Benchmark_Parallel_JSON_Serialization(b *testing.B) {
	data := BenchmarkData{
		TxID:      "tx-bench-parallel",
		AccountID: "acc-user-enterprise-parallel",
		Amount:    150.25,
		Timestamp: 1789123456,
		IsAudit:   false,
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(&data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
