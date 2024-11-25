package analyzer

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

// ThreadState represents the state of a thread
type ThreadState string

const (
	RUNNABLE      ThreadState = "RUNNABLE"
	BLOCKED       ThreadState = "BLOCKED"
	WAITING       ThreadState = "WAITING"
	TIMED_WAITING ThreadState = "TIMED_WAITING"
	UNKNOWN       ThreadState = "UNKNOWN"
)

// Thread represents a single thread in the thread dump
type Thread struct {
	Name       string
	ID         string
	Tid        string
	Priority   int
	State      ThreadState
	Daemon     bool
	StackTrace []string
	LockInfo   *LockInfo
}

// LockInfo contains information about locks held or waited for by a thread
type LockInfo struct {
	LockName     string
	LockOwnerID  string
	IsWaitingFor bool
}

// ThreadDumpAnalysis contains the analysis results of a thread dump
type ThreadDumpAnalysis struct {
	Threads              []*Thread
	ThreadsByState       map[ThreadState][]*Thread
	DeadlockFound        bool
	TotalThreads         int
	StateCount           map[ThreadState]int
	GCAnalysis           *GCThreadAnalysis
	DaemonThreadCount    int
	NonDaemonThreadCount int
	ThreadPools          []*ThreadPool
	DeadlockChains       [][]*Thread
}

// ThreadPool represents a thread pool in the thread dump
type ThreadPool struct {
	Name          string
	ActiveThreads int
	CoreSize      int
	MaxSize       int
	Threads       []*Thread
}

// Regular expressions for parsing thread dump
var (
	// Format 1: "Thread-7" prio=4 tid=0x0b482220 nid=0x1570 in Object.wait() [0x0bbcf000..0x0bbcfae8]
	threadHeaderRegex1 = regexp.MustCompile(`^"([^"]+)"\s+(?:daemon\s+)?prio=(\d+)\s+tid=([^\s]+)\s+nid=[^\s]+\s+(?:in\s+)?([^\[]+)`)

	// Format 2: "Reference Handler" #2 daemon prio=10 os_prio=2 tid=0x00007f8de4009000 nid=0x6b03 waiting on condition [0x00007f8de4d06000]
	threadHeaderRegex2 = regexp.MustCompile(`^"([^"]+)"\s+#(\d+)\s+(?:daemon\s+)?.*?tid=([^\s]+)\s+.*?\[([^\]]+)\]`)

	// Format 3: "main" #1 prio=5 os_prio=0 cpu=15.63ms elapsed=96.27s tid=0x000001c51eef5000 nid=0x2760 waiting on condition  [0x000001c51ee9f000]
	threadHeaderRegex3 = regexp.MustCompile(`^"([^"]+)"|#(\d+).*?(?:state=(\w+)|waiting on condition)\s+\[([^\]]+)\]`)

	// State patterns
	waitingOnPattern = regexp.MustCompile(`(?i)waiting on|parking to wait|waiting for|waiting to lock`)
	sleepingPattern  = regexp.MustCompile(`(?i)sleeping|timed waiting`)
	blockedPattern   = regexp.MustCompile(`(?i)blocked`)
	runnablePattern  = regexp.MustCompile(`(?i)runnable`)
)

func parseInt(s string) int {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0
	}
	return n
}

func determineStateFromDesc(desc string) ThreadState {
	desc = strings.ToLower(desc)
	switch {
	case strings.Contains(desc, "runnable"):
		return RUNNABLE
	case strings.Contains(desc, "blocked"):
		return BLOCKED
	case strings.Contains(desc, "waiting"):
		return WAITING
	case strings.Contains(desc, "timed_waiting") || strings.Contains(desc, "sleeping"):
		return TIMED_WAITING
	default:
		return UNKNOWN
	}
}

func (a *ThreadDumpAnalysis) addThread(thread *Thread) {
	if thread.State == "" {
		thread.State = UNKNOWN
	}

	// Add thread to the main list
	a.Threads = append(a.Threads, thread)

	// Update total thread count
	a.TotalThreads++

	// Update state counts
	if a.ThreadsByState[thread.State] == nil {
		a.ThreadsByState[thread.State] = make([]*Thread, 0)
	}
	a.ThreadsByState[thread.State] = append(a.ThreadsByState[thread.State], thread)
	a.StateCount[thread.State]++

	// Update daemon/non-daemon counts
	if thread.Daemon {
		a.DaemonThreadCount++
	} else {
		a.NonDaemonThreadCount++
	}

	// Log thread details for debugging
	log.Printf("Added thread: %s, State: %s, Daemon: %v, Stack trace lines: %d",
		thread.Name, thread.State, thread.Daemon, len(thread.StackTrace))
}

func AnalyzeThreadDump(filePath string) (*ThreadDumpAnalysis, error) {
	log.Printf("Starting analysis of file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	analysis := &ThreadDumpAnalysis{
		ThreadsByState: make(map[ThreadState][]*Thread),
		StateCount:     make(map[ThreadState]int),
		ThreadPools:    make([]*ThreadPool, 0),
		Threads:        make([]*Thread, 0),
	}

	log.Printf("Initialized analysis struct")

	scanner := bufio.NewScanner(file)
	var currentThread *Thread
	var stackTrace []string
	lineCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++

		if line == "" || strings.HasPrefix(line, "Full thread dump") {
			continue
		}

		// Try all thread header formats
		var thread *Thread
		if matches := threadHeaderRegex1.FindStringSubmatch(line); matches != nil {
			log.Printf("Found thread header (format 1): %s", line)
			thread = &Thread{
				Name:       matches[1],
				Priority:   parseInt(matches[2]),
				Tid:        matches[3],
				State:      determineStateFromDesc(matches[4]),
				Daemon:     strings.Contains(strings.ToLower(line), "daemon"),
				StackTrace: make([]string, 0),
			}
		} else if matches := threadHeaderRegex2.FindStringSubmatch(line); matches != nil {
			log.Printf("Found thread header (format 2): %s", line)
			thread = &Thread{
				Name:       matches[1],
				ID:         matches[2],
				State:      determineStateFromDesc(matches[3]),
				Daemon:     strings.Contains(strings.ToLower(line), "daemon"),
				StackTrace: make([]string, 0),
			}
		} else if matches := threadHeaderRegex3.FindStringSubmatch(line); matches != nil {
			log.Printf("Found thread header (format 3): %s", line)
			name := matches[1]
			if name == "" {
				name = matches[2]
			}
			state := matches[3]
			if state == "" {
				state = matches[4]
			}
			thread = &Thread{
				Name:       name,
				State:      ThreadState(strings.ToUpper(state)),
				StackTrace: make([]string, 0),
			}
		}

		if thread != nil {
			log.Printf("Processing new thread: %s (State: %s)", thread.Name, thread.State)
			// Save previous thread if exists
			if currentThread != nil {
				currentThread.StackTrace = stackTrace
				analysis.addThread(currentThread)
				log.Printf("Added previous thread to analysis: %s (Stack trace lines: %d)",
					currentThread.Name, len(currentThread.StackTrace))
			}

			currentThread = thread
			stackTrace = make([]string, 0)
			continue
		}

		// Add line to stack trace if we have a current thread
		if currentThread != nil {
			// Look for additional state information in the stack trace
			if currentThread.State == "" || currentThread.State == UNKNOWN {
				if waitingOnPattern.MatchString(line) {
					currentThread.State = WAITING
					log.Printf("Found state in stack trace for thread %s: WAITING", currentThread.Name)
				} else if sleepingPattern.MatchString(line) {
					currentThread.State = TIMED_WAITING
					log.Printf("Found state in stack trace for thread %s: TIMED_WAITING", currentThread.Name)
				} else if blockedPattern.MatchString(line) {
					currentThread.State = BLOCKED
					log.Printf("Found state in stack trace for thread %s: BLOCKED", currentThread.Name)
				} else if runnablePattern.MatchString(line) {
					currentThread.State = RUNNABLE
					log.Printf("Found state in stack trace for thread %s: RUNNABLE", currentThread.Name)
				}
			}
			stackTrace = append(stackTrace, line)
		}
	}

	// Add the last thread if there is one
	if currentThread != nil {
		currentThread.StackTrace = stackTrace
		analysis.addThread(currentThread)
		log.Printf("Added final thread to analysis: %s (Stack trace lines: %d)",
			currentThread.Name, len(currentThread.StackTrace))
	}

	log.Printf("Finished initial thread parsing. Total lines processed: %d", lineCount)
	log.Printf("Analysis stats - Total threads: %d, Daemon: %d, Non-daemon: %d",
		analysis.TotalThreads, analysis.DaemonThreadCount, analysis.NonDaemonThreadCount)
	log.Printf("Thread states: %v", analysis.StateCount)

	// Analyze thread pools
	analysis.analyzeThreadPools()

	// Analyze deadlocks
	analysis.analyzeDeadlocks()

	// Analyze GC threads
	gcAnalysis, err := AnalyzeGCThreads(analysis.Threads)
	if err != nil {
		return nil, fmt.Errorf("error analyzing GC threads: %v", err)
	}
	analysis.GCAnalysis = gcAnalysis

	log.Printf("Analysis completed successfully")
	log.Printf("GC Analysis: %+v", analysis.GCAnalysis)
	return analysis, nil
}

func (a *ThreadDumpAnalysis) analyzeThreadPools() {
	poolMap := make(map[string][]*Thread)

	// Group threads by their pool names
	for _, thread := range a.Threads {
		if strings.Contains(thread.Name, "pool") {
			poolName := extractPoolName(thread.Name)
			poolMap[poolName] = append(poolMap[poolName], thread)
		}
	}

	// Create ThreadPool objects
	for name, threads := range poolMap {
		pool := &ThreadPool{
			Name:          name,
			ActiveThreads: len(threads),
			CoreSize:      extractPoolSize(name, "core"),
			MaxSize:       extractPoolSize(name, "max"),
			Threads:       threads,
		}
		a.ThreadPools = append(a.ThreadPools, pool)
	}
}

func extractPoolName(threadName string) string {
	// Extract pool name from thread name (e.g., "pool-1-thread-1" -> "pool-1")
	parts := strings.Split(threadName, "-")
	if len(parts) >= 2 {
		return parts[0] + "-" + parts[1]
	}
	return threadName
}

func extractPoolSize(poolName string, sizeType string) int {
	// This is a placeholder implementation
	// In a real application, you might want to get this information from the thread dump
	// or from the application configuration
	return 10 // Default size
}

func (a *ThreadDumpAnalysis) analyzeDeadlocks() {
	// Simple deadlock detection: look for cycles in lock dependencies
	lockGraph := make(map[string]string) // thread ID -> waiting for thread ID

	for _, thread := range a.Threads {
		if thread.LockInfo != nil && thread.LockInfo.IsWaitingFor && thread.LockInfo.LockOwnerID != "" {
			lockGraph[thread.ID] = thread.LockInfo.LockOwnerID
		}
	}

	// Check for cycles using Floyd's cycle-finding algorithm
	for id := range lockGraph {
		slow := id
		fast := id

		for {
			if slow == "" || fast == "" {
				break
			}

			slow = lockGraph[slow]
			if slow == "" {
				break
			}

			fast = lockGraph[fast]
			if fast == "" {
				break
			}
			fast = lockGraph[fast]
			if fast == "" {
				break
			}

			if slow == fast {
				a.DeadlockFound = true
				return
			}
		}
	}
}
