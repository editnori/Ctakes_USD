package dictionary

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// CTAKESIntegration provides integration with the cTAKES dictionary creator
type CTAKESIntegration struct {
	ctakesHome string
	javaHome   string
	logger     *BuildLogger
}

// NewCTAKESIntegration creates a new cTAKES integration
func NewCTAKESIntegration(logger *BuildLogger) (*CTAKESIntegration, error) {
	integration := &CTAKESIntegration{
		logger: logger,
	}

	// Find cTAKES installation
	ctakesHome := integration.findCTAKESHome()
	if ctakesHome == "" {
		return nil, fmt.Errorf("cTAKES installation not found")
	}
	integration.ctakesHome = ctakesHome

	// Find Java installation
	javaHome := integration.findJavaHome()
	if javaHome == "" {
		return nil, fmt.Errorf("Java installation not found (requires JDK 17+)")
	}
	integration.javaHome = javaHome

	return integration, nil
}

// findCTAKESHome locates the cTAKES installation directory
func (c *CTAKESIntegration) findCTAKESHome() string {
	// Check common locations
	candidates := []string{
		"apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0",
		filepath.Join("..", "apache-ctakes-6.0.0-bin", "apache-ctakes-6.0.0"),
		filepath.Join(os.Getenv("HOME"), "ctakes"),
		filepath.Join(os.Getenv("HOME"), "apache-ctakes-6.0.0"),
		os.Getenv("CTAKES_HOME"),
	}

	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		// Check if it's a valid cTAKES installation
		libPath := filepath.Join(dir, "lib")
		binPath := filepath.Join(dir, "bin")
		if fi, err := os.Stat(libPath); err == nil && fi.IsDir() {
			if fi, err := os.Stat(binPath); err == nil && fi.IsDir() {
				if c.logger != nil {
					c.logger.Info(fmt.Sprintf("Found cTAKES at: %s", dir), -1)
				}
				return dir
			}
		}
	}

	return ""
}

// findJavaHome locates the Java installation
func (c *CTAKESIntegration) findJavaHome() string {
	// Check JAVA_HOME first
	if javaHome := os.Getenv("JAVA_HOME"); javaHome != "" {
		if c.validateJavaVersion(javaHome) {
			return javaHome
		}
	}

	// Try to find Java in PATH
	javaBin, err := exec.LookPath("java")
	if err == nil {
		// Try to resolve to JAVA_HOME
		javaHome := filepath.Dir(filepath.Dir(javaBin))
		if c.validateJavaVersion(javaHome) {
			return javaHome
		}
	}

	// Check common installation locations
	candidates := []string{
		"/usr/lib/jvm/java-17-openjdk",
		"/usr/lib/jvm/java-17-openjdk-amd64",
		"/usr/lib/jvm/java-18-openjdk",
		"/usr/lib/jvm/java-18-openjdk-amd64",
		"C:\\Program Files\\Java\\jdk-17",
		"C:\\Program Files\\Java\\jdk-18",
	}

	for _, dir := range candidates {
		if c.validateJavaVersion(dir) {
			return dir
		}
	}

	return ""
}

// validateJavaVersion checks if the Java installation is version 17 or higher
func (c *CTAKESIntegration) validateJavaVersion(javaHome string) bool {
	javaBin := filepath.Join(javaHome, "bin", "java")
	if runtime.GOOS == "windows" {
		javaBin += ".exe"
	}

	cmd := exec.Command(javaBin, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Parse version from output
	outputStr := string(output)
	if strings.Contains(outputStr, "version \"17") ||
		strings.Contains(outputStr, "version \"18") ||
		strings.Contains(outputStr, "version \"19") ||
		strings.Contains(outputStr, "version \"20") ||
		strings.Contains(outputStr, "version \"21") {
		return true
	}

	return false
}

// RunDictionaryCreator runs the cTAKES GUI dictionary creator
func (c *CTAKESIntegration) RunDictionaryCreator() error {
	if c.logger != nil {
		c.logger.StartStage("ctakes_gui")
		c.logger.Info("Launching cTAKES Dictionary Creator GUI", -1)
	}

	// Build classpath
	libDir := filepath.Join(c.ctakesHome, "lib")
	classPath := filepath.Join(libDir, "*")

	// Build Java command
	javaBin := filepath.Join(c.javaHome, "bin", "java")
	if runtime.GOOS == "windows" {
		javaBin += ".exe"
	}

	args := []string{
		"-cp", classPath,
		"-Xms512M",
		"-Xmx3g",
		"org.apache.ctakes.gui.dictionary.DictionaryCreator",
	}

	cmd := exec.Command(javaBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if c.logger != nil {
		c.logger.Info(fmt.Sprintf("Executing: %s %s", javaBin, strings.Join(args, " ")), -1)
	}

	return cmd.Run()
}

// BuildDictionaryHeadless builds a dictionary using cTAKES in headless mode
func (c *CTAKESIntegration) BuildDictionaryHeadless(config *Config, umlsPath, outputPath string,
	progressCallback func(stage, message string, progress float64)) error {

	if c.logger != nil {
		c.logger.StartStage("ctakes_headless")
		c.logger.Info("Starting headless dictionary build with cTAKES", 0.0)
	}

	// Build classpath
	libDir := filepath.Join(c.ctakesHome, "lib")
	classPath := filepath.Join(libDir, "*")

	// Prepare memory settings
	initialHeap := "512M"
	maxHeap := "3g"
	if config.Memory.InitialHeapMB > 0 {
		initialHeap = fmt.Sprintf("%dM", config.Memory.InitialHeapMB)
	}
	if config.Memory.MaxHeapMB > 0 {
		maxHeap = fmt.Sprintf("%dM", config.Memory.MaxHeapMB)
	}

	// Build Java command for headless build
	javaBin := filepath.Join(c.javaHome, "bin", "java")
	if runtime.GOOS == "windows" {
		javaBin += ".exe"
	}

	// Create properties file for configuration
	propsFile, err := c.createPropertiesFile(config, umlsPath, outputPath)
	if err != nil {
		return fmt.Errorf("failed to create properties file: %w", err)
	}
	defer os.Remove(propsFile)

	args := []string{
		"-cp", classPath,
		fmt.Sprintf("-Xms%s", initialHeap),
		fmt.Sprintf("-Xmx%s", maxHeap),
		"-Djava.awt.headless=true",
		"org.apache.ctakes.gui.dictionary.DictionaryBuilder",
		"-p", propsFile,
	}

	cmd := exec.Command(javaBin, args...)

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cTAKES: %w", err)
	}

	// Monitor output
	go c.monitorOutput(stdout, progressCallback, false)
	go c.monitorOutput(stderr, progressCallback, true)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("cTAKES build failed: %w", err)
	}

	if c.logger != nil {
		c.logger.Info("cTAKES dictionary build completed", 1.0)
	}

	return nil
}

// createPropertiesFile creates a properties file for cTAKES configuration
func (c *CTAKESIntegration) createPropertiesFile(config *Config, umlsPath, outputPath string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "ctakes-dict-*.properties")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write properties
	props := []string{
		fmt.Sprintf("umls.dir=%s", umlsPath),
		fmt.Sprintf("output.dir=%s", outputPath),
		fmt.Sprintf("dictionary.name=%s", config.Name),
		fmt.Sprintf("dictionary.description=%s", config.Description),
	}

	// Add semantic types
	if len(config.SemanticTypes) > 0 {
		props = append(props, fmt.Sprintf("semantic.types=%s", strings.Join(config.SemanticTypes, ",")))
	}

	// Add vocabularies
	if len(config.Vocabularies) > 0 {
		props = append(props, fmt.Sprintf("vocabularies=%s", strings.Join(config.Vocabularies, ",")))
	}

	// Add languages
	if len(config.Languages) > 0 {
		props = append(props, fmt.Sprintf("languages=%s", strings.Join(config.Languages, ",")))
	}

	// Add term types
	if len(config.TermTypes) > 0 {
		props = append(props, fmt.Sprintf("term.types=%s", strings.Join(config.TermTypes, ",")))
	}

	// Add filters
	props = append(props,
		fmt.Sprintf("min.term.length=%d", config.Filters.MinTermLength),
		fmt.Sprintf("max.term.length=%d", config.Filters.MaxTermLength),
		fmt.Sprintf("exclude.suppressible=%t", config.Filters.ExcludeSuppressible),
		fmt.Sprintf("exclude.obsolete=%t", config.Filters.ExcludeObsolete),
		fmt.Sprintf("case.sensitive=%t", config.Filters.CaseSensitive),
		fmt.Sprintf("use.normalization=%t", config.Filters.UseNormalization),
	)

	// Add output formats
	props = append(props,
		fmt.Sprintf("output.bsv=%t", true), // BSV is always generated
		fmt.Sprintf("output.hsqldb=%t", config.Outputs.BuildHSQLDB),
		fmt.Sprintf("output.lucene=%t", config.Outputs.BuildLucene),
		fmt.Sprintf("output.tsv=%t", config.Outputs.EmitTSV),
		fmt.Sprintf("output.jsonl=%t", config.Outputs.EmitJSONL),
	)

	// Write to file
	for _, prop := range props {
		if _, err := fmt.Fprintln(tmpFile, prop); err != nil {
			return "", err
		}
	}

	return tmpFile.Name(), nil
}

// monitorOutput monitors command output and reports progress
func (c *CTAKESIntegration) monitorOutput(pipe io.ReadCloser, progressCallback func(stage, message string, progress float64), isError bool) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		// Log the output
		if c.logger != nil {
			if isError {
				c.logger.Warning(line, nil)
			} else {
				c.logger.Info(line, -1)
			}
		}

		// Parse progress if callback provided
		if progressCallback != nil && !isError {
			progress := c.parseProgress(line)
			if progress >= 0 {
				progressCallback("ctakes", line, progress)
			} else {
				progressCallback("ctakes", line, -1)
			}
		}
	}
}

// parseProgress attempts to parse progress from cTAKES output
func (c *CTAKESIntegration) parseProgress(line string) float64 {
	// Look for progress indicators in the output
	if strings.Contains(line, "Processing") {
		// Try to extract percentage
		if strings.Contains(line, "%") {
			// Parse percentage from line
			parts := strings.Split(line, "%")
			if len(parts) > 0 {
				// Find the last number before %
				for i := len(parts[0]) - 1; i >= 0; i-- {
					if parts[0][i] < '0' || parts[0][i] > '9' {
						if i < len(parts[0])-1 {
							var progress float64
							if _, err := fmt.Sscanf(parts[0][i+1:], "%f", &progress); err == nil {
								return progress / 100.0
							}
						}
						break
					}
				}
			}
		}
	}

	// Check for stage indicators
	if strings.Contains(line, "Reading UMLS") {
		return 0.1
	} else if strings.Contains(line, "Processing concepts") {
		return 0.3
	} else if strings.Contains(line, "Building index") {
		return 0.6
	} else if strings.Contains(line, "Writing output") {
		return 0.8
	} else if strings.Contains(line, "Complete") || strings.Contains(line, "Finished") {
		return 1.0
	}

	return -1
}

// GetCTAKESVersion returns the cTAKES version
func (c *CTAKESIntegration) GetCTAKESVersion() string {
	// Try to read version from cTAKES installation
	versionFile := filepath.Join(c.ctakesHome, "RELEASE_NOTES.html")
	if data, err := os.ReadFile(versionFile); err == nil {
		content := string(data)
		if strings.Contains(content, "6.0.0") {
			return "6.0.0"
		} else if strings.Contains(content, "5.") {
			return "5.x"
		} else if strings.Contains(content, "4.") {
			return "4.x"
		}
	}

	return "Unknown"
}

// ValidateCTAKESInstallation validates the cTAKES installation
func (c *CTAKESIntegration) ValidateCTAKESInstallation() error {
	// Check required directories
	requiredDirs := []string{"bin", "lib", "resources", "desc"}
	for _, dir := range requiredDirs {
		path := filepath.Join(c.ctakesHome, dir)
		if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
			return fmt.Errorf("missing required directory: %s", dir)
		}
	}

	// Check for key JAR files
	libDir := filepath.Join(c.ctakesHome, "lib")
	requiredJars := []string{
		"ctakes-core-",
		"ctakes-dictionary-lookup-",
		"ctakes-gui-",
		"uimaj-core-",
	}

	entries, err := os.ReadDir(libDir)
	if err != nil {
		return fmt.Errorf("cannot read lib directory: %w", err)
	}

	for _, required := range requiredJars {
		found := false
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), required) && strings.HasSuffix(entry.Name(), ".jar") {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("missing required JAR: %s*.jar", required)
		}
	}

	// Check Java version
	if !c.validateJavaVersion(c.javaHome) {
		return fmt.Errorf("Java 17 or higher is required")
	}

	return nil
}

// ---------- Pipeline Discovery & Runner ----------

// PiperFile represents a discoverable cTAKES pipeline (.piper)
type PiperFile struct {
	Name        string
	Path        string
	Category    string
	Description string
}

// DiscoverPipelines scans the cTAKES resources for available .piper files
func (c *CTAKESIntegration) DiscoverPipelines() ([]PiperFile, error) {
	resourcesDir := filepath.Join(c.ctakesHome, "resources", "org", "apache", "ctakes")
	var piperFiles []PiperFile

	// Walk resources directory and collect .piper files
	err := filepath.Walk(resourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".piper") {
			category := categorizePiperPath(path)
			name := strings.TrimSuffix(info.Name(), ".piper")
			piperFiles = append(piperFiles, PiperFile{
				Name:     name,
				Path:     path,
				Category: category,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by category then name for stable UI display
	sort.Slice(piperFiles, func(i, j int) bool {
		if piperFiles[i].Category == piperFiles[j].Category {
			return strings.ToLower(piperFiles[i].Name) < strings.ToLower(piperFiles[j].Name)
		}
		return strings.ToLower(piperFiles[i].Category) < strings.ToLower(piperFiles[j].Category)
	})

	return piperFiles, nil
}

func categorizePiperPath(path string) string {
	lowered := strings.ToLower(path)
	switch {
	case strings.Contains(lowered, string(filepath.Separator)+"clinical"+string(filepath.Separator)):
		return "Clinical"
	case strings.Contains(lowered, string(filepath.Separator)+"temporal"+string(filepath.Separator)):
		return "Temporal"
	case strings.Contains(lowered, string(filepath.Separator)+"relation"+string(filepath.Separator)):
		return "Relation"
	case strings.Contains(lowered, string(filepath.Separator)+"coreference"+string(filepath.Separator)):
		return "Coreference"
	case strings.Contains(lowered, string(filepath.Separator)+"dictionary"+string(filepath.Separator)):
		return "Dictionary"
	case strings.Contains(lowered, string(filepath.Separator)+"chunker"+string(filepath.Separator)):
		return "Chunker"
	case strings.Contains(lowered, string(filepath.Separator)+"core"+string(filepath.Separator)):
		return "Core"
	case strings.Contains(lowered, string(filepath.Separator)+"examples"+string(filepath.Separator)):
		return "Examples"
	default:
		return "Other"
	}
}

// BuildClassPath builds the classpath used to run Piper pipelines without shell scripts
func (c *CTAKESIntegration) BuildClassPath() string {
	// Match setenv.bat compositing
	// desc and resources are directories, config/* and lib/* are globs that java resolves with wildcard
	desc := filepath.Join(c.ctakesHome, "desc")
	resources := filepath.Join(c.ctakesHome, "resources")
	configStar := filepath.Join(c.ctakesHome, "config", "*")
	libStar := filepath.Join(c.ctakesHome, "lib", "*")
	sep := string(os.PathListSeparator)
	return strings.Join([]string{desc, resources, configStar, libStar}, sep)
}

// PipelineRunConfig configures a Piper pipeline run
type PipelineRunConfig struct {
	PiperFile   string
	InputDir    string
	OutputDir   string
	XMIDir      string   // optional; if empty uses OutputDir
	LookupXml   string   // path to dictionary xml; used by fast lookup pipelines via -l
	UMLSKey     string   // optional umls key; passed as --key
	InitialHeap string   // e.g. "512M"
	MaxHeap     string   // e.g. "3g"
	ExtraArgs   []string // additional CLI args, e.g., custom cli flags exposed by piper
}

// RunPiper executes a .piper pipeline using the PiperFileRunner
func (c *CTAKESIntegration) RunPiper(cfg PipelineRunConfig, progress func(stage, message string, progress float64)) error {
	if cfg.PiperFile == "" {
		return errors.New("PiperFile is required")
	}
	if cfg.InputDir == "" || cfg.OutputDir == "" {
		return errors.New("InputDir and OutputDir are required")
	}

	javaBin := filepath.Join(c.javaHome, "bin", "java")
	if runtime.GOOS == "windows" {
		javaBin += ".exe"
	}

	xms := cfg.InitialHeap
	if xms == "" {
		xms = "512M"
	}
	xmx := cfg.MaxHeap
	if xmx == "" {
		xmx = "3g"
	}

	classPath := c.BuildClassPath()

	// Piper runner class
	runner := "org.apache.ctakes.core.pipeline.PiperFileRunner"

	// Build args
	args := []string{
		"-cp", classPath,
		"-Xms" + xms,
		"-Xmx" + xmx,
		runner,
		"-p", cfg.PiperFile,
		"-i", cfg.InputDir,
		"-o", cfg.OutputDir,
	}
	if cfg.XMIDir != "" {
		args = append(args, "--xmiOut", cfg.XMIDir)
	}
	if cfg.LookupXml != "" {
		// fast dictionary subpipe declares cli LookupXml=l
		args = append(args, "-l", cfg.LookupXml)
	}
	if cfg.UMLSKey != "" {
		args = append(args, "--key", cfg.UMLSKey)
	}
	if len(cfg.ExtraArgs) > 0 {
		args = append(args, cfg.ExtraArgs...)
	}

	cmd := exec.Command(javaBin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if c.logger != nil {
		c.logger.StartStage("pipeline")
		c.logger.Info(fmt.Sprintf("Running Piper: %s %s", javaBin, strings.Join(args, " ")), 0)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Piper: %w", err)
	}

	go c.monitorOutput(stdout, progress, false)
	go c.monitorOutput(stderr, progress, true)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pipeline failed: %w", err)
	}
	if c.logger != nil {
		c.logger.Info("Pipeline completed", 1.0)
	}
	return nil
}
