package secretsmanager

type Client interface {
	Get(secret_id string) (string, error)
}
