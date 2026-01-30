package scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/binary-gws/agent/internal/collector"
	"github.com/binary-gws/agent/internal/logging"
	"github.com/binary-gws/agent/internal/platform"
	"github.com/binary-gws/agent/internal/transport"
)

type Config struct {
	UUID             string
	ClientID         string
	SiteID           string
	Platform         *platform.Info
	HeartbeatSeconds int
	Collector        *collector.Collector
	Transport        *transport.Client
	Logger           *logging.Logger
	Version          string
	Commit           string
	BuildDate        string
}

type Payload struct {
	BatchIndex      int             `json:"batch_index"`
	PayloadVersion  string          `json:"payload_version"`
	UUID            string          `json:"uuid"`
	ClientID        string          `json:"client_id"`
	SiteID          string          `json:"site_id"`
	Stats           Stats           `json:"stats"`
	Additional      Additional      `json:"additional"`
	AgentTimestamp  string          `json:"agent_timestamp_utc,omitempty"`
}

type Stats struct {
	SystemStatus collector.SystemStatus  `json:"system_status"`
	Compute      *collector.ComputeMetrics `json:"compute,omitempty"`
}

type Additional struct {
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	Platform     string `json:"platform"`
	AgentVersion string `json:"agent_version,omitempty"`
	Build        string `json:"build,omitempty"`
}

type Scheduler struct {
	config              Config
	consecutiveFailures int
	lastSuccessAt       *time.Time
	batchCounter        int
}

func New(cfg Config) *Scheduler {
	return &Scheduler{
		config: cfg,
	}
}

func (s *Scheduler) buildPayload() *Payload {
	s.batchCounter++
	payload := &Payload{
		BatchIndex:     s.batchCounter,
		PayloadVersion: "1.0",
		UUID:           s.config.UUID,
		ClientID:       s.config.ClientID,
		SiteID:         s.config.SiteID,
		Stats: Stats{
			SystemStatus: s.config.Collector.GetSystemStatus(),
			Compute:      s.config.Collector.GetComputeMetrics(false),
		},
		Additional: Additional{
			Metadata: Metadata{
				Platform: s.config.Platform.Platform,
			},
		},
		AgentTimestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if s.config.Version != "" {
		payload.Additional.Metadata.AgentVersion = s.config.Version
	}
	if s.config.Commit != "" || s.config.BuildDate != "" {
		buildInfo := ""
		if s.config.Commit != "" {
			buildInfo = s.config.Commit
		}
		if s.config.BuildDate != "" {
			if buildInfo != "" {
				buildInfo += " "
			}
			buildInfo += s.config.BuildDate
		}
		payload.Additional.Metadata.Build = buildInfo
	}

	return payload
}

func (s *Scheduler) SendOnce(ctx context.Context, dryRun bool) error {
	payload := s.buildPayload()

	if dryRun {
		jsonData, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		s.config.Logger.Info("Dry run payload", map[string]interface{}{
			"payload": string(jsonData),
		})
		return nil
	}

	err := s.config.Transport.SendHeartbeat(ctx, payload, transport.DefaultRetryConfig, nil)
	if err != nil {
		s.consecutiveFailures++
		s.config.Logger.Error("Failed to send heartbeat", map[string]interface{}{
			"error":               err.Error(),
			"consecutive_failures": s.consecutiveFailures,
		})
		return err
	}

	now := time.Now()
	s.lastSuccessAt = &now
	s.consecutiveFailures = 0
	s.config.Logger.Info("Heartbeat sent successfully", map[string]interface{}{
		"last_success_at": s.lastSuccessAt.UTC().Format(time.RFC3339),
	})
	return nil
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(s.config.HeartbeatSeconds) * time.Second)
	defer ticker.Stop()

	if err := s.SendOnce(ctx, false); err != nil {
		s.config.Logger.Warn("Initial heartbeat failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	for {
		select {
		case <-ctx.Done():
			s.config.Logger.Info("Scheduler stopping", nil)
			return ctx.Err()
		case <-ticker.C:
			if err := s.SendOnce(ctx, false); err != nil {
				s.config.Logger.Debug("Heartbeat cycle failed", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}
