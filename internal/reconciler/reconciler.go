package reconciler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/veerendra2/komodo-secrets-sync/pkg/komodo"
	"github.com/veerendra2/komodo-secrets-sync/pkg/secrets"
)

type Config struct {
	Interval time.Duration `name:"interval" help:"Reconcile interval" env:"INTERVAL" default:"5m"`
}

type Reconciler struct {
	cfg      Config
	smClient secrets.Client
	kClient  komodo.Client

	secretsCache map[string]string
}

func (r *Reconciler) Run(ctx context.Context) error {
	slog.Info("Starting reconciliation loop", "interval", r.cfg.Interval)

	// Initial reconciliation at startup
	if err := r.reconcile(ctx); err != nil {
		slog.Error("Initial reconciliation failed", "error", err)
	}

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Reconciliation stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := r.reconcile(ctx); err != nil {
				slog.Error("Reconciliation failed", "error", err)
			}
		}
	}
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get all secrets
	dump, err := r.smClient.Dump(ctx)
	if err != nil {
		return err
	}

	// Find changed secrets by comparing hash
	var modified []secrets.Secret
	for _, secret := range dump.Secrets {
		currentHash := hash(secret.Value)
		cachedHash, exists := r.secretsCache[secret.Key]

		if !exists || cachedHash != currentHash {
			modified = append(modified, secret)
			r.secretsCache[secret.Key] = currentHash
		}
	}

	slog.Info("Reconciliation scan", "total", len(dump.Secrets), "modified", len(modified))

	// Sync modified secrets to Komodo
	syncTime := time.Now().UTC()
	for _, secret := range modified {
		syncMsg := fmt.Sprintf("Synced by komodo-secrets-sync at %s", syncTime.Format(time.RFC3339))
		err := r.kClient.UpsertVariable(ctx, secret.Key, secret.Value, syncMsg, true)
		if err != nil {
			slog.Error("Failed to sync secret", "key", secret.Key, "error", err)
			continue
		}
		slog.Debug("Secret synced", "key", secret.Key)
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
		secretsCache: make(map[string]string),
	}
}
