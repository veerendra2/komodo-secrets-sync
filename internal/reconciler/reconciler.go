package reconciler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/veerendra2/komodo-secrets-sync/pkg/komodo"
	"github.com/veerendra2/komodo-secrets-sync/pkg/secrets"
)

type Config struct {
	Interval time.Duration `name:"interval" help:"Reconcile interval" env:"INTERVAL" default:"5m"`
	Timeout  time.Duration `name:"timeout" help:"Reconcile timeout" env:"TIMEOUT" default:"1m"`
}

type Reconciler struct {
	cfg      Config
	smClient secrets.Client
	kClient  komodo.Client

	secretsCache sync.Map
}

func (r *Reconciler) Run(ctx context.Context) error {
	slog.Info("Starting reconciliation loop",
		"interval", r.cfg.Interval.String(),
		"timeout", r.cfg.Timeout.String(),
	)

	// Run initial reconciliation immediately on startup
	if err := r.reconcile(ctx); err != nil {
		slog.Error("Initial reconciliation failed", "error", err)
	}

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Reconciliation stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := r.reconcile(ctx); err != nil {
				slog.Error("Reconciliation failed", "error", err)
			}
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Timeout)
	defer cancel()

	secretsCollection, err := r.smClient.FetchAll(ctx)
	if err != nil {
		return err
	}

	// Find changed secrets by comparing hash
	var modified []secrets.Secret
	for _, secret := range secretsCollection.Secrets {
		currentHash := hash(secret.Value)

		// Load cached hash from sync.Map
		cachedValue, exists := r.secretsCache.Load(secret.Key)
		if !exists {
			// Secret doesn't exist in cache, it's new
			modified = append(modified, secret)
			r.secretsCache.Store(secret.Key, currentHash)
		} else {
			// Secret exists, compare hashes
			cachedHash, ok := cachedValue.(string)
			if !ok || cachedHash != currentHash {
				modified = append(modified, secret)
				r.secretsCache.Store(secret.Key, currentHash)
			}
		}
	}

	slog.Info("Reconciliation scan", "total", len(secretsCollection.Secrets), "modified", len(modified))

	syncTime := time.Now().UTC()
	successCount := 0

	for _, secret := range modified {
		syncMsg := fmt.Sprintf("Synced by komodo-secrets-sync at %s", syncTime.Format(time.RFC3339))
		err := r.kClient.UpsertVariable(ctx, secret.Key, secret.Value, syncMsg, true)
		if err != nil {
			slog.Error("Failed to sync secret", "key", secret.Key, "error", err)
			continue
		}
		successCount++
		slog.Debug("Secret synced", "key", secret.Key)
	}

	if len(modified) > 0 {
		slog.Info("Sync completed", "synced", successCount, "failed", len(modified)-successCount)
	}

	return nil
}

func hash(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

func New(cfg Config, smClient secrets.Client, kClient komodo.Client) *Reconciler {
	return &Reconciler{
		cfg:          cfg,
		smClient:     smClient,
		kClient:      kClient,
		secretsCache: sync.Map{},
	}
}
