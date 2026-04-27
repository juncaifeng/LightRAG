package server

import (
	"context"
	"log"
	"sync"
	"time"
)

// ServiceCache provides an in-memory cache backed by SQLite persistence.
type ServiceCache struct {
	mu        sync.RWMutex
	instances map[string][]*ServiceInstanceInfo // service_name → instances
	store     *ServiceStore
}

// NewServiceCache creates a cache backed by the given store.
func NewServiceCache(store *ServiceStore) *ServiceCache {
	return &ServiceCache{
		instances: make(map[string][]*ServiceInstanceInfo),
		store:     store,
	}
}

// Register writes to DB and updates cache.
func (c *ServiceCache) Register(inst *ServiceInstanceInfo) error {
	if err := c.store.RegisterInstance(inst); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	list := c.instances[inst.ServiceName]
	// Replace existing instance with same ID, or append
	for i, existing := range list {
		if existing.InstanceID == inst.InstanceID {
			list[i] = inst
			c.instances[inst.ServiceName] = list
			return nil
		}
	}
	c.instances[inst.ServiceName] = append(list, inst)
	return nil
}

// Heartbeat updates the heartbeat timestamp in DB and cache.
func (c *ServiceCache) Heartbeat(serviceName, instanceID string, ttlSeconds int32) error {
	expiresAt := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	if err := c.store.UpdateHeartbeat(serviceName, instanceID, expiresAt); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	list := c.instances[serviceName]
	for _, inst := range list {
		if inst.InstanceID == instanceID {
			inst.LastHeartbeat = time.Now()
			inst.ExpiresAt = expiresAt
			return nil
		}
	}
	return nil
}

// Unregister removes from DB and cache.
func (c *ServiceCache) Unregister(serviceName, instanceID string) error {
	if err := c.store.UnregisterInstance(serviceName, instanceID); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	list := c.instances[serviceName]
	for i, inst := range list {
		if inst.InstanceID == instanceID {
			c.instances[serviceName] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return nil
}

// List returns cached instances for a service (or all if serviceName is empty).
func (c *ServiceCache) List(serviceName string) []*ServiceInstanceInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if serviceName != "" {
		result := make([]*ServiceInstanceInfo, 0)
		for _, inst := range c.instances[serviceName] {
			if time.Now().Before(inst.ExpiresAt) {
				result = append(result, inst)
			}
		}
		return result
	}

	// All services
	result := make([]*ServiceInstanceInfo, 0)
	now := time.Now()
	for _, list := range c.instances {
		for _, inst := range list {
			if now.Before(inst.ExpiresAt) {
				result = append(result, inst)
			}
		}
	}
	return result
}

// ReloadFromDB loads all instances from DB into cache. Called on startup.
func (c *ServiceCache) ReloadFromDB() error {
	instances, err := c.store.ListInstances("")
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.instances = make(map[string][]*ServiceInstanceInfo)
	for _, inst := range instances {
		c.instances[inst.ServiceName] = append(c.instances[inst.ServiceName], inst)
	}
	log.Printf("Service cache reloaded: %d instances across %d services", len(instances), len(c.instances))
	return nil
}

// StartCleanupLoop runs a background goroutine that periodically removes expired instances.
func (c *ServiceCache) StartCleanupLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := c.store.CleanExpired()
				if err != nil {
					log.Printf("Service cache cleanup error: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("Service cache: cleaned %d expired instances", n)
					// Reload cache from DB to stay in sync
					if err := c.ReloadFromDB(); err != nil {
						log.Printf("Service cache reload error: %v", err)
					}
				}
			}
		}
	}()
}
