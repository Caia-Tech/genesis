package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// SafeGoroutine runs a function in a goroutine with panic recovery
func SafeGoroutine(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("🚨 Panic in %s goroutine: %v\n", name, r)
				
				// Print stack trace for debugging
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				fmt.Printf("Stack trace:\n%s\n", buf[:n])
			}
		}()
		
		fn()
	}()
}

// CheckMemoryUsage prints current memory statistics
func CheckMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	fmt.Printf("📊 Memory Usage: Alloc=%d KB, TotalAlloc=%d KB, Sys=%d KB, NumGC=%d\n",
		bToKb(m.Alloc), bToKb(m.TotalAlloc), bToKb(m.Sys), m.NumGC)
}

func bToKb(b uint64) uint64 {
	return b / 1024
}

// Helper math functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GracefulShutdown provides a way to handle cleanup during shutdown
type GracefulShutdown struct {
	cleanupFuncs []func()
	timeout      time.Duration
}

func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		cleanupFuncs: make([]func(), 0),
		timeout:      timeout,
	}
}

func (gs *GracefulShutdown) AddCleanup(fn func()) {
	gs.cleanupFuncs = append(gs.cleanupFuncs, fn)
}

func (gs *GracefulShutdown) Shutdown() {
	fmt.Println("🔄 Starting graceful shutdown...")
	
	done := make(chan struct{})
	go func() {
		for i, cleanup := range gs.cleanupFuncs {
			fmt.Printf("🧹 Running cleanup %d/%d\n", i+1, len(gs.cleanupFuncs))
			cleanup()
		}
		close(done)
	}()
	
	select {
	case <-done:
		fmt.Println("✅ Graceful shutdown completed")
	case <-time.After(gs.timeout):
		fmt.Println("⚠️  Shutdown timeout exceeded, forcing exit")
		os.Exit(1)
	}
}

// ResourceMonitor tracks system resources to prevent overloads
type ResourceMonitor struct {
	maxMemoryMB   int
	checkInterval time.Duration
	shutdown      chan bool
}

func NewResourceMonitor(maxMemoryMB int, checkInterval time.Duration) *ResourceMonitor {
	return &ResourceMonitor{
		maxMemoryMB:   maxMemoryMB,
		checkInterval: checkInterval,
		shutdown:      make(chan bool, 1),
	}
}

func (rm *ResourceMonitor) Start() {
	SafeGoroutine("resource-monitor", func() {
		ticker := time.NewTicker(rm.checkInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-rm.shutdown:
				return
			case <-ticker.C:
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				
				currentMemoryMB := int(m.Alloc / 1024 / 1024)
				if currentMemoryMB > rm.maxMemoryMB {
					fmt.Printf("🚨 MEMORY ALERT: Using %d MB > %d MB limit! Running GC...\n", 
						currentMemoryMB, rm.maxMemoryMB)
					runtime.GC()
					
					// Check again after GC
					runtime.ReadMemStats(&m)
					newMemoryMB := int(m.Alloc / 1024 / 1024)
					if newMemoryMB > rm.maxMemoryMB {
						fmt.Printf("🆘 CRITICAL: Memory still high after GC: %d MB\n", newMemoryMB)
					}
				}
			}
		}
	})
}

func (rm *ResourceMonitor) Stop() {
	select {
	case rm.shutdown <- true:
	default:
	}
}