# cTAKES Integration Plan

## Overview
Apache cTAKES is a Java-based system, so we need to bridge between our Go TUI and the Java runtime. Here are the proposed approaches:

## Integration Approaches

### 1. **Java Process Management (Recommended)**
- Launch cTAKES as a subprocess from Go
- Communicate via stdin/stdout or files
- Benefits: Direct control, no additional services needed
- Challenges: Process lifecycle management, error handling

```go
// Example structure
type CTakesProcess struct {
    cmd *exec.Cmd
    stdin io.WriteCloser
    stdout io.ReadCloser
    stderr io.ReadCloser
}
```

### 2. **REST API Wrapper**
- Create a Java Spring Boot wrapper around cTAKES
- Expose REST endpoints for processing
- Go TUI communicates via HTTP
- Benefits: Clean separation, scalable
- Challenges: Additional service to maintain

### 3. **File-Based Processing**
- Write input files to a watched directory
- cTAKES processes files and writes output
- Go TUI monitors output directory
- Benefits: Simple, reliable
- Challenges: Latency, file management

### 4. **JNI/CGO Bridge** (Not Recommended)
- Direct Java-Go integration
- Complex and platform-specific

## Proposed Implementation Steps

### Phase 1: Basic Integration
1. Create Java wrapper class for cTAKES CLI
2. Implement process spawning in Go
3. Set up communication protocol (JSON over stdout)
4. Handle process lifecycle

### Phase 2: Pipeline Configuration
1. Create configuration files for different pipelines
2. Implement pipeline builder in Java
3. Pass configuration from TUI to Java process

### Phase 3: Advanced Features
1. Batch processing
2. Result caching
3. Dictionary management
4. Custom annotators

## Java Wrapper Structure

```
ctakes-wrapper/
├── src/main/java/
│   ├── CTakesWrapper.java      # Main entry point
│   ├── PipelineBuilder.java    # Configure pipelines
│   ├── DocumentProcessor.java  # Process documents
│   └── ResultFormatter.java    # Format output as JSON
├── pom.xml                      # Maven configuration
└── scripts/
    └── run.sh                   # Launch script
```

## Communication Protocol

### Request Format (JSON)
```json
{
  "id": "uuid",
  "action": "analyze",
  "text": "Patient presents with...",
  "pipeline": "default",
  "options": {
    "outputFormat": "json",
    "includeConfidence": true
  }
}
```

### Response Format (JSON)
```json
{
  "id": "uuid",
  "status": "success",
  "results": {
    "entities": [
      {
        "type": "DiseaseDisorderMention",
        "text": "hypertension",
        "begin": 20,
        "end": 32,
        "polarity": "positive",
        "umls": ["C0020538"]
      }
    ],
    "medications": [...],
    "procedures": [...]
  }
}
```

## Directory Structure for Integration

```
ctakes-tui/
├── cmd/                    # Go command-line tools
├── internal/
│   ├── ctakes/            # cTAKES integration
│   │   ├── process.go     # Process management
│   │   ├── client.go      # Communication client
│   │   └── parser.go      # Result parsing
│   └── models/            # Data models
├── java/                  # Java wrapper code
│   └── ctakes-wrapper/
└── resources/             # cTAKES resources
    ├── dictionaries/
    └── models/
```

## Configuration

### config.yaml
```yaml
ctakes:
  javaBin: "java"
  heapSize: "2G"
  wrapperJar: "./java/ctakes-wrapper/target/ctakes-wrapper.jar"
  resources: "./resources"
  timeout: 30s
  
pipelines:
  default:
    components:
      - sentenceDetector
      - tokenizer
      - posTagger
      - chunker
      - dictionaryLookup
      - assertion
  
  minimal:
    components:
      - sentenceDetector
      - tokenizer
      - dictionaryLookup
```

## Error Handling

1. **Process Crashes**: Automatic restart with backoff
2. **Timeout**: Configurable timeout per request
3. **Memory Issues**: Monitor and restart if needed
4. **Invalid Input**: Validate before sending to cTAKES

## Testing Strategy

1. **Unit Tests**: Mock cTAKES responses
2. **Integration Tests**: Test with real cTAKES process
3. **Performance Tests**: Measure throughput and latency
4. **Stress Tests**: Handle multiple concurrent requests

## Next Steps

1. **Verify cTAKES Installation**: Check if cTAKES is available
2. **Create Minimal Java Wrapper**: Start with basic text analysis
3. **Implement Process Manager**: Handle Java process lifecycle
4. **Add Communication Layer**: JSON over stdout/stdin
5. **Test Integration**: Verify end-to-end flow

## Resources

- [cTAKES Developer Guide](https://cwiki.apache.org/confluence/display/CTAKES/cTAKES+4.0+Developer+Guide)
- [cTAKES API Documentation](https://ctakes.apache.org/apidocs/)
- [UIMA Framework](https://uima.apache.org/) (underlying framework)