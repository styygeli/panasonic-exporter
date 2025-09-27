// panasonic-exporter is a Prometheus exporter for Panasonic breaker box energy data.
package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Configuration is loaded from environment variables.
var (
	breakerBoxURL string
	powerMappings map[string]int
)

const (
	listenAddress = ":9190"
	namespace     = "panasonic"
)

// panasonicCollector manages all logic for fetching data and creating metrics.
type panasonicCollector struct {
	powerDesc *prometheus.Desc
	mutex     sync.Mutex
}

// newPanasonicCollector initializes the collector.
func newPanasonicCollector() *panasonicCollector {
	return &panasonicCollector{
		powerDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "power", "watts"),
			"Current power consumption in Watts.",
			[]string{"entity", "friendly_name"},
			nil,
		),
	}
}

// Describe implements the prometheus.Collector interface.
func (c *panasonicCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.powerDesc
}

// Collect implements the prometheus.Collector interface.
// It is triggered by Prometheus on each scrape.
func (c *panasonicCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	resp, err := http.Get(breakerBoxURL)
	if err != nil {
		log.Printf("Error fetching data from breaker box: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: Received non-200 status code: %s", resp.Status)
		return
	}

	// The CSV parser is configured to be flexible, as device-generated files
	// can have an inconsistent number of columns per row.
	reader := csv.NewReader(resp.Body)
	reader.FieldsPerRecord = -1 // Allow variable number of fields per record

	records, err := reader.ReadAll()
	if err != nil {
		log.Printf("Error parsing CSV data: %v", err)
		return
	}

	var dataRow []string
	headerIndex := -1

	// To handle malformed or partial responses, we search for the specific header
	// row ("YYYYMMDDhhmm") and assume the data is on the next line.
	for i, row := range records {
		if len(row) > 0 && row[0] == "YYYYMMDDhhmm" {
			headerIndex = i
			break
		}
	}

	if headerIndex == -1 {
		log.Printf("Error: CSV header row ('YYYYMMDDhhmm') not found in the response.")
		return
	}
	if len(records) <= headerIndex+1 {
		log.Printf("Error: Data row not found immediately after the header row.")
		return
	}

	dataRow = records[headerIndex+1]

	// Iterate through our configured circuit mappings to create metrics.
	for key, columnIndex := range powerMappings {
		if len(dataRow) <= columnIndex {
			log.Printf("Warning: column index %d for entity '%s' is out of bounds.", columnIndex, key)
			continue
		}

		value, err := strconv.ParseInt(dataRow[columnIndex], 16, 64)
		if err != nil {
			log.Printf("Warning: could not parse hex value for entity '%s': %v", key, err)
			continue
		}

		// Certain circuits require a multiplier.
		if key == "main" || key == "ecocute" {
			value *= 10
		}

		friendlyName := strings.ReplaceAll(strings.Title(strings.ReplaceAll(key, "_", " ")), " ", "")
		ch <- prometheus.MustNewConstMetric(c.powerDesc, prometheus.GaugeValue, float64(value), key, friendlyName)
	}
}

func main() {
	// Load configuration from a .env file in the same directory as the executable.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on existing environment variables.")
	}

	breakerBoxURL = os.Getenv("PANASONIC_URL")
	mappingsJSON := os.Getenv("PANASONIC_MAPPINGS")

	if breakerBoxURL == "" || mappingsJSON == "" {
		log.Fatal("Error: PANASONIC_URL and PANASONIC_MAPPINGS must be set in the .env file or environment.")
	}

	// Parse the circuit mappings from the JSON string.
	if err := json.Unmarshal([]byte(mappingsJSON), &powerMappings); err != nil {
		log.Fatalf("Error: Could not parse PANASONIC_MAPPINGS JSON: %v", err)
	}

	collector := newPanasonicCollector()
	prometheus.MustRegister(collector)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html><head><title>Panasonic Exporter</title></head>
			<body><h1>Panasonic Breaker Box Exporter</h1><p><a href="/metrics">Metrics</a></p></body>
			</html>
		`))
	})

	log.Printf("Exporter starting. Listening on address %s", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatalf("Error: Could not start HTTP server: %v", err)
	}
}
