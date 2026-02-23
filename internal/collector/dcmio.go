package collector

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// DCMIOMetrics holds business metrics fetched from DCMIO's PostgreSQL.
// Fields are embedded in Stats and appear flat in the JSON payload.
type DCMIOMetrics struct {
	ImagesProcessedCurrent   int64 `json:"images_processed_current"`
	ImagesSyncedCurrent      int64 `json:"images_synced_current"`
	ImagesSyncPendingCurrent int64 `json:"images_sync_pending_current"`
	TasksPendingCurrent      int64 `json:"tasks_pending_current"`
}

// DCMIOCollector queries DCMIO's postgres and caches results.
// It supports two connection modes:
//
//   - Direct TCP (containerName == ""): connects using postgresURL.
//   - Docker exec (containerName != ""): shells out via `docker exec` to reach
//     a postgres container that has no exposed port. Requires the docker CLI to
//     be on PATH and the process to have access to the Docker socket.
type DCMIOCollector struct {
	// Direct TCP connection mode
	postgresURL string

	// Docker exec mode
	containerName string
	postgresUser  string
	postgresPass  string
	postgresDB    string

	intervalSeconds int
	windowHours     int

	mu            sync.Mutex
	db            *sql.DB // only used in direct TCP mode
	lastMetrics   *DCMIOMetrics
	lastFetchTime time.Time
}

// NewDCMIOCollector creates a collector that connects directly to postgres via TCP.
func NewDCMIOCollector(postgresURL string, intervalSeconds, windowHours int) *DCMIOCollector {
	return &DCMIOCollector{
		postgresURL:     postgresURL,
		intervalSeconds: intervalSeconds,
		windowHours:     windowHours,
	}
}

// NewDCMIODockerCollector creates a collector that queries postgres through
// `docker exec`. Use this when the postgres container has no exposed port.
func NewDCMIODockerCollector(containerName, pgUser, pgPass, pgDB string, intervalSeconds, windowHours int) *DCMIOCollector {
	return &DCMIOCollector{
		containerName:   containerName,
		postgresUser:    pgUser,
		postgresPass:    pgPass,
		postgresDB:      pgDB,
		intervalSeconds: intervalSeconds,
		windowHours:     windowHours,
	}
}

func (d *DCMIOCollector) ensureConnected() error {
	if d.db != nil {
		if err := d.db.Ping(); err == nil {
			return nil
		}
		d.db.Close()
		d.db = nil
	}

	db, err := sql.Open("postgres", d.postgresURL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("ping postgres: %w", err)
	}

	d.db = db
	return nil
}

// GetMetrics returns cached metrics or fetches fresh ones if the cache expired.
func (d *DCMIOCollector) GetMetrics() *DCMIOMetrics {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	if d.lastMetrics != nil && now.Sub(d.lastFetchTime) < time.Duration(d.intervalSeconds)*time.Second {
		return d.lastMetrics
	}

	var metrics *DCMIOMetrics
	var err error

	if d.containerName != "" {
		metrics, err = d.fetchMetricsDockerExec()
	} else {
		if err = d.ensureConnected(); err == nil {
			metrics, err = d.fetchMetricsDirect()
		}
	}

	if err != nil || metrics == nil {
		// Return stale cache on error rather than dropping metrics.
		return d.lastMetrics
	}

	d.lastMetrics = metrics
	d.lastFetchTime = now
	return d.lastMetrics
}

// fetchMetricsDirect runs all queries via a direct postgres connection.
//
// Queries explained:
//
//	scans_delivered — graphs where the publish task completed successfully within
//	the configured window. This is the gateway equivalent of "images processed":
//	a DICOM series was uploaded to the cloud, AI inference ran, and the result
//	was published back to the hospital PACS.
//
//	scans_pending — graphs that still have at least one pending task (status=0)
//	within the window. These are studies currently in flight.
//
//	tasks_pending — individual pending tasks within the window. Finer-grained
//	than scans_pending; used for the tasks_pending_current field.
func (d *DCMIOCollector) fetchMetricsDirect() (*DCMIOMetrics, error) {
	const query = `
		SELECT
			(
				SELECT COUNT(DISTINCT graph_id)
				FROM job_manager_task
				WHERE type LIKE '%publish%'
				  AND status = 2
				  AND updated_at >= NOW() - ($1 * INTERVAL '1 hour')
			) AS scans_delivered,

			(
				SELECT COUNT(DISTINCT graph_id)
				FROM job_manager_task
				WHERE status = 0
				  AND created_at >= NOW() - ($1 * INTERVAL '1 hour')
			) AS scans_pending,

			(
				SELECT COUNT(*)
				FROM job_manager_task
				WHERE status = 0
				  AND created_at >= NOW() - ($1 * INTERVAL '1 hour')
			) AS tasks_pending
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var scansDelivered, scansPending, tasksPending int64
	row := d.db.QueryRowContext(ctx, query, d.windowHours)
	if err := row.Scan(&scansDelivered, &scansPending, &tasksPending); err != nil {
		return nil, fmt.Errorf("scan dcmio metrics: %w", err)
	}

	return &DCMIOMetrics{
		ImagesProcessedCurrent:   scansDelivered,
		ImagesSyncedCurrent:      scansDelivered,
		ImagesSyncPendingCurrent: scansPending,
		TasksPendingCurrent:      tasksPending,
	}, nil
}

// fetchMetricsDockerExec runs the SQL via `docker exec <container> psql ...`.
// The SQL window hours are interpolated directly (integer, not user input).
// Output format: three comma-separated integers on a single line.
func (d *DCMIOCollector) fetchMetricsDockerExec() (*DCMIOMetrics, error) {
	sql := fmt.Sprintf(`SELECT `+
		`(SELECT COUNT(DISTINCT graph_id) FROM job_manager_task WHERE type LIKE '%%publish%%' AND status = 2 AND updated_at >= NOW() - INTERVAL '%d hours') AS scans_delivered,`+
		`(SELECT COUNT(DISTINCT graph_id) FROM job_manager_task WHERE status = 0 AND created_at >= NOW() - INTERVAL '%d hours') AS scans_pending,`+
		`(SELECT COUNT(*) FROM job_manager_task WHERE status = 0 AND created_at >= NOW() - INTERVAL '%d hours') AS tasks_pending`,
		d.windowHours, d.windowHours, d.windowHours,
	)

	args := []string{"exec"}
	if d.postgresPass != "" {
		args = append(args, "-e", "PGPASSWORD="+d.postgresPass)
	}
	args = append(args,
		d.containerName,
		"psql", "-U", d.postgresUser, "-d", d.postgresDB,
		"-t", "-A", "-F", ",",
		"-c", sql,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "docker", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("docker exec psql: %w", err)
	}

	line := strings.TrimSpace(string(out))
	parts := strings.Split(line, ",")
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected psql output: %q", line)
	}

	scansDelivered, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse scans_delivered: %w", err)
	}
	scansPending, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse scans_pending: %w", err)
	}
	tasksPending, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse tasks_pending: %w", err)
	}

	return &DCMIOMetrics{
		ImagesProcessedCurrent:   scansDelivered,
		ImagesSyncedCurrent:      scansDelivered,
		ImagesSyncPendingCurrent: scansPending,
		TasksPendingCurrent:      tasksPending,
	}, nil
}
