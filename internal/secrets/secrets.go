package secrets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type WorkspaceSecret struct {
	ID             string     `json:"id"`
	AgentID        string     `json:"agent_id"`
	Hash           string     `json:"hash"`
	Path           string     `json:"path"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	MaxAttempts    int        `json:"max_attempts"`
	FailedAttempts int        `json:"failed_attempts"`
}

type SecretApproval struct {
	AgentID    string    `json:"agent_id"`
	UserID     string    `json:"user_id"`
	Path       string    `json:"path"`
	ApprovedAt time.Time `json:"approved_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

var (
	secrets     = make(map[string]*WorkspaceSecret)
	secretsMu   sync.RWMutex
	approvals   = make(map[string]*SecretApproval)
	approvalsMu sync.RWMutex
)

func GetSecretsDir() string {
	return filepath.Join(config.GetConfigDir(), "secrets")
}

func LoadSecrets() error {
	dir := GetSecretsDir()
	data, err := os.ReadFile(filepath.Join(dir, "secrets.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var loaded []WorkspaceSecret
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	secretsMu.Lock()
	defer secretsMu.Unlock()
	for i := range loaded {
		secrets[loaded[i].ID] = &loaded[i]
	}

	approvalsData, err := os.ReadFile(filepath.Join(dir, "approvals.json"))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		var loadedApprovals []SecretApproval
		if err := json.Unmarshal(approvalsData, &loadedApprovals); err != nil {
			return err
		}
		for i := range loadedApprovals {
			ap := &loadedApprovals[i]
			approvals[getApprovalKey(ap.AgentID, ap.UserID, ap.Path)] = ap
		}
	}

	return nil
}

func SaveSecrets() error {
	dir := GetSecretsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	secretsMu.RLock()
	all := make([]WorkspaceSecret, 0, len(secrets))
	for _, s := range secrets {
		all = append(all, *s)
	}
	secretsMu.RUnlock()

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "secrets.json"), data, 0644); err != nil {
		return err
	}

	approvalsMu.RLock()
	allApprovals := make([]SecretApproval, 0, len(approvals))
	for _, a := range approvals {
		allApprovals = append(allApprovals, *a)
	}
	approvalsMu.RUnlock()

	approvalData, err := json.MarshalIndent(allApprovals, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "approvals.json"), approvalData, 0644)
}

func AddSecret(agentID, secretKey, path string, maxAttempts int) error {
	hash := hashKey(secretKey)

	secretsMu.Lock()
	defer secretsMu.Unlock()

	id := fmt.Sprintf("secret_%d", time.Now().Unix())

	secrets[id] = &WorkspaceSecret{
		ID:             id,
		AgentID:        agentID,
		Hash:           hash,
		Path:           path,
		CreatedAt:      time.Now(),
		MaxAttempts:    maxAttempts,
		FailedAttempts: 0,
	}

	return SaveSecrets()
}

func VerifySecret(agentID, secretKey, path string) (bool, error) {
	secretsMu.RLock()
	defer secretsMu.RUnlock()

	inputHash := hashKey(secretKey)

	for _, s := range secrets {
		if s.AgentID == agentID && s.Path == path {
			if s.Hash == inputHash {
				return true, nil
			}
			s.FailedAttempts++
			return false, fmt.Errorf("invalid key (attempt %d/%d)", s.FailedAttempts, s.MaxAttempts)
		}
	}

	return false, fmt.Errorf("no secret configured for this path")
}

func ApproveAccess(agentID, userID, path string, duration time.Duration) error {
	approval := &SecretApproval{
		AgentID:    agentID,
		UserID:     userID,
		Path:       path,
		ApprovedAt: time.Now(),
		ExpiresAt:  time.Now().Add(duration),
	}

	approvalsMu.Lock()
	defer approvalsMu.Unlock()

	key := getApprovalKey(agentID, userID, path)
	approvals[key] = approval

	return SaveSecrets()
}

func CheckApproval(agentID, userID, path string) bool {
	approvalsMu.RLock()
	defer approvalsMu.RUnlock()

	key := getApprovalKey(agentID, userID, path)
	if ap, ok := approvals[key]; ok {
		return time.Now().Before(ap.ExpiresAt)
	}
	return false
}

func RevokeApproval(agentID, userID, path string) error {
	approvalsMu.Lock()
	defer approvalsMu.Unlock()

	key := getApprovalKey(agentID, userID, path)
	delete(approvals, key)

	return SaveSecrets()
}

func GetSecretPath(agentID string) string {
	return filepath.Join(GetSecretsDir(), agentID)
}

func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func getApprovalKey(agentID, userID, path string) string {
	return fmt.Sprintf("%s:%s:%s", agentID, userID, path)
}
