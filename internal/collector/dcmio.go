package collector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// PendingTask represents a single pending task from job_manager_task.
// Mirrors the fields returned by the original Python PENDING_QUERY.
type PendingTask struct {
	ID        int64  `json:"id"`
	GraphID   string `json:"graph_id"`
	Type      string `json:"type"`
	Priority  int    `json:"priority"`
	CreatedAt string `json:"created_at"`
}

// DCMIOMetrics holds business metrics fetched from DCMIO's PostgreSQL.
// Fields are embedded in Stats and appear flat in the JSON payload.
// Field names match what the Pulse dashboard expects:
//
//	tasks_pending_current       → "Tasks Pending"
//	images_synced_current       → "Scans Delivered last 12 hrs"
//	images_sync_pending_current → "Failed Uploads"
//	total_scans_current         → "Total Scans" (also denominator for Success Rate)
//	pending_tasks               → list of pending task details
type DCMIOMetrics struct {
	TasksPendingCurrent      int64         `json:"tasks_pending_current"`
	ImagesSyncedCurrent      int64         `json:"images_synced_current"`
	ImagesSyncPendingCurrent int64         `json:"images_sync_pending_current"`
	TotalScansCurrent        int64         `json:"total_scans_current"`
	PendingTasks             []PendingTask `json:"pending_tasks,omitempty"`
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
// Queries mirror the original Python monitoring script:
//
//	pending_tasks_count — all pending tasks (status=0) created within the window.
//
//	completed_scans — distinct graph IDs where any task completed (status=2)
//	within the window. Represents studies processed by the system.
//
//	failed_uploads — failed upload tasks (status=-1,
//	type='workflow_manager.tasks.upload.upload') within the window.
//
//	total_scans — unique graph IDs whose first task was created within the window.
func (d *DCMIOCollector) fetchMetricsDirect() (*DCMIOMetrics, error) {
	const query = `
		SELECT
			(
				SELECT COUNT(*)
				FROM job_manager_task
				WHERE status = 0
				  AND created_at >= NOW() - ($1 * INTERVAL '1 hour')
			) AS pending_tasks_count,

			(
				SELECT COUNT(DISTINCT graph_id)
				FROM job_manager_task
				WHERE status = 2
				  AND updated_at >= NOW() - ($1 * INTERVAL '1 hour')
			) AS completed_scans,

			(
				SELECT COUNT(*)
				FROM job_manager_task
				WHERE status = -1
				  AND type = 'workflow_manager.tasks.upload.upload'
				  AND created_at >= NOW() - ($1 * INTERVAL '1 hour')
			) AS failed_uploads,

			(
				SELECT COUNT(*)
				FROM (
					SELECT graph_id
					FROM job_manager_task
					GROUP BY graph_id
					HAVING MIN(created_at) >= NOW() - ($1 * INTERVAL '1 hour')
				) t
			) AS total_scans
	`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var pendingTasksCount, completedScans, failedUploads, totalScans int64
	row := d.db.QueryRowContext(ctx, query, d.windowHours)
	if err := row.Scan(&pendingTasksCount, &completedScans, &failedUploads, &totalScans); err != nil {
		return nil, fmt.Errorf("scan dcmio metrics: %w", err)
	}

	metrics := &DCMIOMetrics{
		TasksPendingCurrent:      pendingTasksCount,
		ImagesSyncedCurrent:      completedScans,
		ImagesSyncPendingCurrent: failedUploads,
		TotalScansCurrent:        totalScans,
	}

	// Fetch pending task details (PENDING_QUERY from original Python script).
	const pendingQuery = `
		SELECT id, graph_id, type, priority, created_at
		FROM job_manager_task
		WHERE status = 0
		  AND created_at >= NOW() - ($1 * INTERVAL '1 hour')
		ORDER BY created_at DESC
	`
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	rows, err := d.db.QueryContext(ctx2, pendingQuery, d.windowHours)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var task PendingTask
			var createdAt time.Time
			if scanErr := rows.Scan(&task.ID, &task.GraphID, &task.Type, &task.Priority, &createdAt); scanErr == nil {
				task.CreatedAt = createdAt.UTC().Format(time.RFC3339)
				metrics.PendingTasks = append(metrics.PendingTasks, task)
			}
		}
	}

	return metrics, nil
}

// fetchMetricsDockerExec runs the SQL via `docker exec <container> psql ...`.
// The SQL window hours are interpolated directly (integer, not user input).
// Output format: four comma-separated integers on a single line.
func (d *DCMIOCollector) fetchMetricsDockerExec() (*DCMIOMetrics, error) {
	sql := fmt.Sprintf(`SELECT `+
		`(SELECT COUNT(*) FROM job_manager_task WHERE status = 0 AND created_at >= NOW() - INTERVAL '%d hours') AS pending_tasks_count,`+
		`(SELECT COUNT(DISTINCT graph_id) FROM job_manager_task WHERE status = 2 AND updated_at >= NOW() - INTERVAL '%d hours') AS completed_scans,`+
		`(SELECT COUNT(*) FROM job_manager_task WHERE status = -1 AND type = 'workflow_manager.tasks.upload.upload' AND created_at >= NOW() - INTERVAL '%d hours') AS failed_uploads,`+
		`(SELECT COUNT(*) FROM (SELECT graph_id FROM job_manager_task GROUP BY graph_id HAVING MIN(created_at) >= NOW() - INTERVAL '%d hours') t) AS total_scans`,
		d.windowHours, d.windowHours, d.windowHours, d.windowHours,
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
	if len(parts) != 4 {
		return nil, fmt.Errorf("unexpected psql output: %q", line)
	}

	pendingTasksCount, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse pending_tasks_count: %w", err)
	}
	completedScans, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse completed_scans: %w", err)
	}
	failedUploads, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse failed_uploads: %w", err)
	}
	totalScans, err := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse total_scans: %w", err)
	}

	metrics := &DCMIOMetrics{
		TasksPendingCurrent:      pendingTasksCount,
		ImagesSyncedCurrent:      completedScans,
		ImagesSyncPendingCurrent: failedUploads,
		TotalScansCurrent:        totalScans,
	}

	// Fetch pending task details via second docker exec (PENDING_QUERY).
	pendingSQL := fmt.Sprintf(
		`SELECT COALESCE(json_agg(row_to_json(t)), '[]'::json) FROM `+
			`(SELECT id, graph_id, type, priority, created_at FROM job_manager_task `+
			`WHERE status = 0 AND created_at >= NOW() - INTERVAL '%d hours' `+
			`ORDER BY created_at DESC) t`,
		d.windowHours,
	)

	pendingArgs := []string{"exec"}
	if d.postgresPass != "" {
		pendingArgs = append(pendingArgs, "-e", "PGPASSWORD="+d.postgresPass)
	}
	pendingArgs = append(pendingArgs,
		d.containerName,
		"psql", "-U", d.postgresUser, "-d", d.postgresDB,
		"-t", "-A",
		"-c", pendingSQL,
	)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	if pendingOut, pendingErr := exec.CommandContext(ctx2, "docker", pendingArgs...).Output(); pendingErr == nil {
		var tasks []PendingTask
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(string(pendingOut))), &tasks); jsonErr == nil {
			metrics.PendingTasks = tasks
		}
	}

	return metrics, nil
}
