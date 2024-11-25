package analyzer

import (
	"log"
	"regexp"
	"strings"
)

// GCThreadType represents different types of GC threads
type GCThreadType int

const (
	GC_UNKNOWN GCThreadType = iota
	GC_TASK
	GC_DAEMON
	GC_CONCURRENT_MARK
	GC_CONCURRENT_SWEEP
	GC_PARALLEL
)

// String returns the string representation of GCThreadType
func (t GCThreadType) String() string {
	switch t {
	case GC_TASK:
		return "GC Task"
	case GC_DAEMON:
		return "GC Daemon"
	case GC_CONCURRENT_MARK:
		return "GC Concurrent Mark"
	case GC_CONCURRENT_SWEEP:
		return "GC Concurrent Sweep"
	case GC_PARALLEL:
		return "GC Parallel"
	default:
		return "Unknown GC Thread"
	}
}

var (
	// Patterns to identify different GC thread types
	gcPatterns = map[*regexp.Regexp]GCThreadType{
		regexp.MustCompile(`(?i)GC task thread`):    GC_TASK,
		regexp.MustCompile(`(?i)GC Daemon`):         GC_DAEMON,
		regexp.MustCompile(`(?i)Concurrent Mark`):   GC_CONCURRENT_MARK,
		regexp.MustCompile(`(?i)Concurrent Sweep`):  GC_CONCURRENT_SWEEP,
		regexp.MustCompile(`(?i)ParallelGC Thread`): GC_PARALLEL,
	}
)

// GCThreadAnalysis contains information about garbage collection threads
type GCThreadAnalysis struct {
	TotalGCThreads int
	GCThreads      []*Thread
	Types          map[string]int
}

// AnalyzeGCThreads analyzes threads related to garbage collection
func AnalyzeGCThreads(threads []*Thread) (*GCThreadAnalysis, error) {
	log.Printf("Starting GC thread analysis")
	analysis := &GCThreadAnalysis{
		Types:     make(map[string]int),
		GCThreads: make([]*Thread, 0),
	}

	for _, thread := range threads {
		if isGCThread(thread) {
			log.Printf("Found GC thread: %s", thread.Name)
			analysis.GCThreads = append(analysis.GCThreads, thread)
			analysis.TotalGCThreads++

			// Categorize GC thread type
			gcType := determineGCType(thread.Name)
			analysis.Types[gcType]++
			log.Printf("Categorized as: %s", gcType)
		}
	}

	log.Printf("GC analysis complete - Total threads: %d, Types: %v",
		analysis.TotalGCThreads, analysis.Types)
	return analysis, nil
}

func isGCThread(thread *Thread) bool {
	name := strings.ToLower(thread.Name)
	stackTrace := strings.ToLower(strings.Join(thread.StackTrace, " "))

	// Check thread name
	isGC := strings.Contains(name, "gc") ||
		strings.Contains(name, "g1") ||
		strings.Contains(name, "concurrent mark") ||
		strings.Contains(name, "cms") ||
		strings.Contains(name, "parallel") ||
		strings.Contains(name, "scavenge")

	// Check stack trace for GC-related methods
	if !isGC {
		isGC = strings.Contains(stackTrace, "gc") ||
			strings.Contains(stackTrace, "garbage collect") ||
			strings.Contains(stackTrace, "markcompact") ||
			strings.Contains(stackTrace, "scavenge") ||
			strings.Contains(stackTrace, "finalize")
	}

	if isGC {
		log.Printf("Thread identified as GC thread: %s", thread.Name)
		if len(thread.StackTrace) > 0 {
			log.Printf("First line of stack trace: %s", thread.StackTrace[0])
		}
	}

	return isGC
}

func determineGCType(threadName string) string {
	name := strings.ToLower(threadName)
	switch {
	case strings.Contains(name, "g1"):
		return "G1 GC"
	case strings.Contains(name, "cms"):
		return "CMS"
	case strings.Contains(name, "concurrent mark"):
		return "Concurrent Mark"
	case strings.Contains(name, "concurrent sweep"):
		return "Concurrent Sweep"
	case strings.Contains(name, "parallel"):
		return "Parallel GC"
	case strings.Contains(name, "scavenge"):
		return "Scavenge"
	case strings.Contains(name, "finalizer"):
		return "Finalizer"
	case strings.Contains(name, "reference"):
		return "Reference Processing"
	default:
		return "Other GC"
	}
}
