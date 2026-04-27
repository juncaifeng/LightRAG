package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// ServiceInstanceInfo represents a registered service instance.
type ServiceInstanceInfo struct {
	ServiceName   string            `json:"service_name"`
	InstanceID    string            `json:"instance_id"`
	Address       string            `json:"address"`
	Version       string            `json:"version"`
	Metadata      map[string]string `json:"metadata"`
	Status        string            `json:"status"`
	RegisteredAt  time.Time         `json:"registered_at"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	ExpiresAt     time.Time         `json:"expires_at"`
}

// ServiceStore provides SQLite persistence for service instances.
type ServiceStore struct {
	db *sql.DB
}

// NewServiceStore opens or creates the SQLite database at dbPath.
func NewServiceStore(dbPath string) (*ServiceStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	return &ServiceStore{db: db}, nil
}

// InitSchema creates the service_instances table if it doesn't exist.
func (s *ServiceStore) InitSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS service_instances (
			service_name   TEXT NOT NULL,
			instance_id    TEXT NOT NULL,
			address        TEXT NOT NULL,
			version        TEXT DEFAULT '',
			metadata       TEXT DEFAULT '{}',
			status         TEXT DEFAULT 'healthy',
			registered_at  INTEGER NOT NULL,
			last_heartbeat INTEGER NOT NULL,
			expires_at     INTEGER NOT NULL,
			PRIMARY KEY (service_name, instance_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	log.Println("Service store schema initialized")
	return nil
}

// RegisterInstance inserts or replaces a service instance.
func (s *ServiceStore) RegisterInstance(inst *ServiceInstanceInfo) error {
	metaJSON, err := json.Marshal(inst.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT OR REPLACE INTO service_instances
			(service_name, instance_id, address, version, metadata, status, registered_at, last_heartbeat, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		inst.ServiceName,
		inst.InstanceID,
		inst.Address,
		inst.Version,
		string(metaJSON),
		inst.Status,
		inst.RegisteredAt.UnixMilli(),
		inst.LastHeartbeat.UnixMilli(),
		inst.ExpiresAt.UnixMilli(),
	)
	return err
}

// UpdateHeartbeat refreshes the heartbeat timestamp and extends expiry.
func (s *ServiceStore) UpdateHeartbeat(serviceName, instanceID string, expiresAt time.Time) error {
	now := time.Now().UnixMilli()
	result, err := s.db.Exec(`
		UPDATE service_instances
		SET last_heartbeat = ?, expires_at = ?
		WHERE service_name = ? AND instance_id = ? AND status = 'healthy'
	`, now, expiresAt.UnixMilli(), serviceName, instanceID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("instance not found: %s/%s", serviceName, instanceID)
	}
	return nil
}

// UnregisterInstance removes a service instance.
func (s *ServiceStore) UnregisterInstance(serviceName, instanceID string) error {
	_, err := s.db.Exec(`
		DELETE FROM service_instances
		WHERE service_name = ? AND instance_id = ?
	`, serviceName, instanceID)
	return err
}

// ListInstances returns all healthy (non-expired) instances, optionally filtered by service name.
func (s *ServiceStore) ListInstances(serviceName string) ([]*ServiceInstanceInfo, error) {
	now := time.Now().UnixMilli()
	var rows *sql.Rows
	var err error

	if serviceName != "" {
		rows, err = s.db.Query(`
			SELECT service_name, instance_id, address, version, metadata, status, registered_at, last_heartbeat, expires_at
			FROM service_instances
			WHERE service_name = ? AND expires_at > ?
			ORDER BY registered_at
		`, serviceName, now)
	} else {
		rows, err = s.db.Query(`
			SELECT service_name, instance_id, address, version, metadata, status, registered_at, last_heartbeat, expires_at
			FROM service_instances
			WHERE expires_at > ?
			ORDER BY service_name, registered_at
		`, now)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*ServiceInstanceInfo
	for rows.Next() {
		inst := &ServiceInstanceInfo{}
		var metaJSON string
		var registeredAt, lastHeartbeat, expiresAt int64
		if err := rows.Scan(
			&inst.ServiceName, &inst.InstanceID, &inst.Address, &inst.Version,
			&metaJSON, &inst.Status, &registeredAt, &lastHeartbeat, &expiresAt,
		); err != nil {
			return nil, err
		}
		inst.RegisteredAt = time.UnixMilli(registeredAt)
		inst.LastHeartbeat = time.UnixMilli(lastHeartbeat)
		inst.ExpiresAt = time.UnixMilli(expiresAt)
		if err := json.Unmarshal([]byte(metaJSON), &inst.Metadata); err != nil {
			inst.Metadata = map[string]string{}
		}
		instances = append(instances, inst)
	}
	return instances, nil
}

// CleanExpired deletes all expired instances and returns the count removed.
func (s *ServiceStore) CleanExpired() (int, error) {
	now := time.Now().UnixMilli()
	result, err := s.db.Exec(`DELETE FROM service_instances WHERE expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// Close closes the database connection.
func (s *ServiceStore) Close() error {
	return s.db.Close()
}
