# Thread Dump Analyzer

A web-based tool for analyzing Java thread dumps. This analyzer helps developers and system administrators understand the state of their Java applications by providing detailed insights into thread states, deadlocks, garbage collection, and thread pools.

## Features

- **Thread State Analysis**
  - Total thread count
  - Thread state distribution (Runnable, Blocked, Waiting, Timed Waiting)
  - Daemon vs Non-daemon thread statistics
  - Detailed stack traces for each thread

- **Garbage Collection Analysis**
  - GC thread identification and categorization
  - Different GC types detection (G1, CMS, Parallel, etc.)
  - GC thread statistics

- **Thread Pool Analysis**
  - Thread pool identification
  - Active threads count
  - Core and max pool sizes
  - Thread pool member details

- **Deadlock Detection**
  - Automatic deadlock detection
  - Visual representation of deadlock chains
  - Lock dependency analysis

## Getting Started

### Prerequisites

- Go 1.16 or later
- Web browser (Chrome, Firefox, Safari, or Edge)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/yourusername/thread-analyzer.git
cd thread-analyzer
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run the application:
```bash
go run main.go
```

4. Open your web browser and navigate to:
```
http://localhost:8080
```

### Usage

1. Prepare your thread dump file (supported formats: .txt, .log)
2. Click "Choose File" on the web interface
3. Select your thread dump file
4. Click "Upload and Analyze"
5. View the analysis results in the following sections:
   - Summary
   - All Threads
   - GC Threads
   - Deadlocks (if any detected)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
