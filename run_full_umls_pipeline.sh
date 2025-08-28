#!/bin/bash

# Advanced cTAKES Pipeline with Full UMLS Dictionary
# This script addresses CuiCodeUtil warnings by ensuring proper dictionary loading

BASE_DIR="/mnt/c/Users/Layth M Qassem/Desktop/CtakesBun"
CTAKES_HOME="${BASE_DIR}/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
OUTPUT_BASE="${BASE_DIR}/output/full_umls_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create output directories
mkdir -p "${OUTPUT_BASE}"
mkdir -p "${OUTPUT_BASE}/logs"
mkdir -p "${OUTPUT_BASE}/metrics"

# Log files
MAIN_LOG="${OUTPUT_BASE}/logs/main_log_${TIMESTAMP}.txt"
ERROR_LOG="${OUTPUT_BASE}/logs/error_log_${TIMESTAMP}.txt"
METRICS_FILE="${OUTPUT_BASE}/metrics/metrics_${TIMESTAMP}.csv"

# Directories to process
DIRECTORIES=(
    "${BASE_DIR}/SD5000_1/EmergencyDepartmentNote"
    "${BASE_DIR}/SD5000_1/InpatientNote"
    "${BASE_DIR}/SD5000_1/OutpatientNote"
    "${BASE_DIR}/SD5000_1/RadiologyReport"
    "${BASE_DIR}/SD5000_1/AdmissionNote"
    "${BASE_DIR}/SD5000_1/DischargeSummary"
)

# Initialize metrics CSV
echo "Directory,FileCount,TotalSizeKB,AvgSizeKB,ProcessingTimeSec,TimePerFileSec,XMIGenerated,Warnings,Errors" > "${METRICS_FILE}"

# Function to setup cTAKES environment
setup_ctakes_env() {
    export CTAKES_HOME="${CTAKES_HOME}"
    
    # Build comprehensive classpath
    CLASS_PATH=""
    
    # Add all JAR files from lib directory
    for jar in ${CTAKES_HOME}/lib/*.jar; do
        CLASS_PATH="${CLASS_PATH}:${jar}"
    done
    
    # Add config JARs (including log4j)
    for jar in ${CTAKES_HOME}/config/*.jar; do
        CLASS_PATH="${CLASS_PATH}:${jar}"
    done
    
    # Add config and resources directories
    CLASS_PATH="${CLASS_PATH}:${CTAKES_HOME}/config:${CTAKES_HOME}/resources"
    
    export CLASS_PATH
    
    # Set up pipeline runner
    export PIPE_RUNNER="org.apache.ctakes.core.pipeline.PiperFileRunner"
    
    # Check for custom dictionary or use default
    if [ -f "${CTAKES_HOME}/resources/org/apache/ctakes/dictionary/lookup/fast/sno_rx_16ab.xml" ]; then
        echo -e "${GREEN}Using SNOMED/RxNorm dictionary${NC}" | tee -a "${MAIN_LOG}"
        export DICT_DESC="${CTAKES_HOME}/resources/org/apache/ctakes/dictionary/lookup/fast/sno_rx_16ab.xml"
    else
        echo -e "${YELLOW}Using default dictionary${NC}" | tee -a "${MAIN_LOG}"
    fi
    
    # Use the full clinical pipeline for comprehensive analysis
    export FAST_PIPER="${CTAKES_HOME}/resources/org/apache/ctakes/clinical/pipeline/DefaultFastPipeline.piper"
}

# Function to count and analyze files
analyze_directory() {
    local dir=$1
    local file_count=0
    local total_size=0
    
    if [ -d "$dir" ]; then
        while IFS= read -r file; do
            ((file_count++))
            size=$(stat -c%s "$file" 2>/dev/null || echo 0)
            ((total_size += size))
        done < <(find "$dir" -name "*.txt" -type f 2>/dev/null)
    fi
    
    echo "${file_count}:${total_size}"
}

# Function to process a single directory
process_directory() {
    local input_dir=$1
    local dir_name=$(basename "$input_dir")
    local output_dir="${OUTPUT_BASE}/${dir_name}_output"
    
    echo "========================================" | tee -a "${MAIN_LOG}"
    echo -e "${GREEN}Processing: ${dir_name}${NC}" | tee -a "${MAIN_LOG}"
    echo "Time: $(date)" | tee -a "${MAIN_LOG}"
    
    # Analyze directory
    local file_info=$(analyze_directory "$input_dir")
    local file_count=$(echo $file_info | cut -d: -f1)
    local total_size=$(echo $file_info | cut -d: -f2)
    
    if [ "$file_count" -eq 0 ]; then
        echo -e "${YELLOW}No text files found in ${input_dir}${NC}" | tee -a "${MAIN_LOG}"
        return
    fi
    
    local total_size_kb=$((total_size / 1024))
    local avg_size_kb=$((total_size_kb / file_count))
    
    echo "  Files to process: ${file_count}" | tee -a "${MAIN_LOG}"
    echo "  Total size: ${total_size_kb} KB" | tee -a "${MAIN_LOG}"
    echo "  Average file size: ${avg_size_kb} KB" | tee -a "${MAIN_LOG}"
    
    # Create output directory
    mkdir -p "${output_dir}"
    
    # Create temporary log for this run
    local temp_log="${OUTPUT_BASE}/logs/${dir_name}_${TIMESTAMP}.log"
    
    # Record start time
    local start_time=$(date +%s)
    
    # Prepare Java command with optimized settings
    local java_cmd="java -cp ${CLASS_PATH} \
        -Xms2g -Xmx8g \
        -XX:+UseG1GC \
        -XX:MaxGCPauseMillis=200 \
        -Dlog4j.configurationFile=${CTAKES_HOME}/config/log4j2.xml \
        -Dctakes.umlsuser=${UMLS_USER:-} \
        -Dctakes.umlspw=${UMLS_PASS:-} \
        ${PIPE_RUNNER} \
        -p ${FAST_PIPER} \
        -i ${input_dir} \
        --xmiOut ${output_dir}"
    
    # Add UMLS key if provided
    if [ ! -z "${UMLS_KEY}" ]; then
        java_cmd="${java_cmd} --key ${UMLS_KEY}"
    fi
    
    # Add custom dictionary if specified
    if [ ! -z "${DICT_DESC}" ]; then
        java_cmd="${java_cmd} -l ${DICT_DESC}"
    fi
    
    echo "  Starting cTAKES processing..." | tee -a "${MAIN_LOG}"
    
    # Run cTAKES and capture output
    cd "${CTAKES_HOME}"
    eval "${java_cmd}" 2>&1 | tee "${temp_log}"
    local exit_code=${PIPESTATUS[0]}
    
    # Record end time
    local end_time=$(date +%s)
    local processing_time=$((end_time - start_time))
    
    # Count output files
    local xmi_count=$(find "$output_dir" -name "*.xmi" 2>/dev/null | wc -l)
    
    # Count warnings and errors
    local warning_count=$(grep -c "WARN" "${temp_log}" 2>/dev/null || echo 0)
    local error_count=$(grep -c "ERROR" "${temp_log}" 2>/dev/null || echo 0)
    
    # Extract CuiCodeUtil warnings
    grep "CuiCodeUtil" "${temp_log}" >> "${ERROR_LOG}" 2>/dev/null
    
    # Calculate metrics
    local time_per_file=0
    if [ "$file_count" -gt 0 ]; then
        time_per_file=$(echo "scale=2; $processing_time / $file_count" | bc)
    fi
    
    # Display results
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✓ Processing completed successfully${NC}" | tee -a "${MAIN_LOG}"
    else
        echo -e "${RED}✗ Processing failed with exit code: ${exit_code}${NC}" | tee -a "${MAIN_LOG}"
    fi
    
    echo "  Processing time: ${processing_time} seconds" | tee -a "${MAIN_LOG}"
    echo "  XMI files generated: ${xmi_count}/${file_count}" | tee -a "${MAIN_LOG}"
    echo "  Time per file: ${time_per_file} seconds" | tee -a "${MAIN_LOG}"
    echo "  Warnings: ${warning_count}" | tee -a "${MAIN_LOG}"
    echo "  Errors: ${error_count}" | tee -a "${MAIN_LOG}"
    
    # Save metrics to CSV
    echo "${dir_name},${file_count},${total_size_kb},${avg_size_kb},${processing_time},${time_per_file},${xmi_count},${warning_count},${error_count}" >> "${METRICS_FILE}"
    
    return $exit_code
}

# Function to generate summary report
generate_summary() {
    local total_time=$1
    
    echo "" | tee -a "${MAIN_LOG}"
    echo "========================================" | tee -a "${MAIN_LOG}"
    echo -e "${GREEN}FINAL SUMMARY${NC}" | tee -a "${MAIN_LOG}"
    echo "========================================" | tee -a "${MAIN_LOG}"
    
    # Calculate totals from metrics file
    local total_files=$(awk -F',' 'NR>1 {sum+=$2} END {print sum}' "${METRICS_FILE}")
    local total_xmi=$(awk -F',' 'NR>1 {sum+=$7} END {print sum}' "${METRICS_FILE}")
    local total_warnings=$(awk -F',' 'NR>1 {sum+=$8} END {print sum}' "${METRICS_FILE}")
    local total_errors=$(awk -F',' 'NR>1 {sum+=$9} END {print sum}' "${METRICS_FILE}")
    
    echo "Total processing time: ${total_time} seconds" | tee -a "${MAIN_LOG}"
    echo "Total files processed: ${total_files}" | tee -a "${MAIN_LOG}"
    echo "Total XMI generated: ${total_xmi}" | tee -a "${MAIN_LOG}"
    echo "Total warnings: ${total_warnings}" | tee -a "${MAIN_LOG}"
    echo "Total errors: ${total_errors}" | tee -a "${MAIN_LOG}"
    
    if [ "$total_files" -gt 0 ]; then
        local avg_time=$(echo "scale=2; $total_time / $total_files" | bc)
        echo "Average time per file: ${avg_time} seconds" | tee -a "${MAIN_LOG}"
    fi
    
    echo "" | tee -a "${MAIN_LOG}"
    echo "Outputs saved to: ${OUTPUT_BASE}" | tee -a "${MAIN_LOG}"
    echo "Metrics file: ${METRICS_FILE}" | tee -a "${MAIN_LOG}"
    echo "Main log: ${MAIN_LOG}" | tee -a "${MAIN_LOG}"
    echo "Error log: ${ERROR_LOG}" | tee -a "${MAIN_LOG}"
}

# Main execution
main() {
    echo "========================================" | tee "${MAIN_LOG}"
    echo "cTAKES Full UMLS Pipeline" | tee -a "${MAIN_LOG}"
    echo "Started: $(date)" | tee -a "${MAIN_LOG}"
    echo "========================================" | tee -a "${MAIN_LOG}"
    
    # Check for UMLS credentials
    echo "" | tee -a "${MAIN_LOG}"
    echo "Checking for UMLS credentials..." | tee -a "${MAIN_LOG}"
    
    if [ -z "${UMLS_KEY}" ]; then
        echo -e "${YELLOW}No UMLS API key found. Set UMLS_KEY environment variable for full dictionary access.${NC}" | tee -a "${MAIN_LOG}"
        echo "You can get a UMLS API key from: https://uts.nlm.nih.gov/uts/profile" | tee -a "${MAIN_LOG}"
        read -p "Enter UMLS API key (or press Enter to continue without): " UMLS_KEY
        export UMLS_KEY
    fi
    
    # Setup environment
    setup_ctakes_env
    
    # Track total processing time
    local total_start=$(date +%s)
    
    # Process each directory
    local success_count=0
    local fail_count=0
    
    for dir in "${DIRECTORIES[@]}"; do
        if [ -d "$dir" ]; then
            if process_directory "$dir"; then
                ((success_count++))
            else
                ((fail_count++))
            fi
        else
            echo -e "${RED}Directory not found: $dir${NC}" | tee -a "${MAIN_LOG}"
            ((fail_count++))
        fi
    done
    
    # Calculate total time
    local total_end=$(date +%s)
    local total_time=$((total_end - total_start))
    
    # Generate summary
    generate_summary $total_time
    
    echo "" | tee -a "${MAIN_LOG}"
    echo "Directories processed successfully: ${success_count}" | tee -a "${MAIN_LOG}"
    echo "Directories failed: ${fail_count}" | tee -a "${MAIN_LOG}"
    echo "Completed: $(date)" | tee -a "${MAIN_LOG}"
    echo "========================================" | tee -a "${MAIN_LOG}"
}

# Run main function
main "$@"