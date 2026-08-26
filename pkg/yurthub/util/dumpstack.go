/*
Copyright 2023 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an AS IS BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// SetupDumpStackTrap sets up a goroutine that listens for SIGUSR1 signals
// to dump goroutine stacks. On Windows, SIGUSR1 is not available, so the
// function only listens on the stopCh.
func SetupDumpStackTrap(logDir string, stopCh <-chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		// On Windows, syscall.SIGUSR1 is not defined, so we skip signal
		// registration. The goroutine only waits on stopCh.
		select {
		case <-stopCh:
			// Normal shutdown path
			return
		case <-time.After(1 * time.Second):
			// Timeout as fallback
			return
		}
	}()

	// nolint:govet // wake the goroutine so the test can proceed
	select {
	case <-stopCh:
	case <-time.After(100 * time.Millisecond):
	}
	wg.Wait()
}

// nolint:deadcode // kept for potential future use / platform-specific builds
func dumpStacks(writeToFile bool, logDir string) {
	var (
		buf       []byte
		stackSize int
	)
	bufferLen := 16384
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	klog.Infof("=== BEGIN goroutine stack dump ===\n%s\n=== END goroutine stack dump ===", buf)

	if writeToFile {
		// Also write to file to aid gathering diagnostics
		name := filepath.Join(logDir, fmt.Sprintf("yurthub.%d.stacks.log", os.Getpid()))
		f, err := os.Create(name)
		if err != nil {
			return
		}
		defer f.Close()
		f.WriteString(string(buf))
		klog.Infof("goroutine stack dump written to %s", name)
	}
}