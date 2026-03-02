package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/allataetm-svg/goclaw/internal/config"
)

type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Source    string                 `json:"source"`
	URL       string                 `json:"url,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type KnowledgeStore struct {
	documents []Document
	index     map[string][]int
	filePath  string
}

func NewKnowledgeStore(agentID string) *KnowledgeStore {
	dir := filepath.Join(config.GetConfigDir(), "memory", "longterm", "knowledge")
	return &KnowledgeStore{
		filePath: filepath.Join(dir, agentID+".json"),
		index:    make(map[string][]int),
	}
}

func (ks *KnowledgeStore) Load() error {
	data, err := os.ReadFile(ks.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			ks.documents = []Document{}
			return nil
		}
		return fmt.Errorf("failed to read knowledge store: %w", err)
	}
	if err := json.Unmarshal(data, &ks.documents); err != nil {
		return fmt.Errorf("failed to parse knowledge store: %w", err)
	}
	ks.rebuildIndex()
	return nil
}

func (ks *KnowledgeStore) Save() error {
	dir := filepath.Dir(ks.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create knowledge dir: %w", err)
	}
	data, err := json.MarshalIndent(ks.documents, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal documents: %w", err)
	}
	if err := os.WriteFile(ks.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write knowledge store: %w", err)
	}
	return nil
}

func (ks *KnowledgeStore) rebuildIndex() {
	ks.index = make(map[string][]int)
	for i, doc := range ks.documents {
		words := extractWords(doc.Content)
		words = append(words, doc.Tags...)
		for _, word := range words {
			ks.index[word] = append(ks.index[word], i)
		}
	}
}

func extractWords(text string) []string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	words := strings.Fields(text)
	var result []string
	seen := make(map[string]bool)
	for _, w := range words {
		if len(w) > 2 && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}
	return result
}

func (ks *KnowledgeStore) AddDocument(doc Document) error {
	if doc.ID == "" {
		hash := sha256.Sum256([]byte(doc.Content + time.Now().String()))
		doc.ID = "doc_" + hex.EncodeToString(hash[:8])
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	ks.documents = append(ks.documents, doc)
	ks.rebuildIndex()
	return ks.Save()
}

func (ks *KnowledgeStore) AddDocumentFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	doc := Document{
		Content: string(data),
		Source:  filepath.Base(path),
	}
	return ks.AddDocument(doc)
}

func (ks *KnowledgeStore) Update(id string, content string) error {
	for i := range ks.documents {
		if ks.documents[i].ID == id {
			ks.documents[i].Content = content
			ks.documents[i].UpdatedAt = time.Now()
			ks.rebuildIndex()
			return ks.Save()
		}
	}
	return fmt.Errorf("document not found: %s", id)
}

func (ks *KnowledgeStore) Delete(id string) error {
	for i := range ks.documents {
		if ks.documents[i].ID == id {
			ks.documents = append(ks.documents[:i], ks.documents[i+1:]...)
			ks.rebuildIndex()
			return ks.Save()
		}
	}
	return fmt.Errorf("document not found: %s", id)
}

func (ks *KnowledgeStore) Get(id string) (*Document, error) {
	for i := range ks.documents {
		if ks.documents[i].ID == id {
			return &ks.documents[i], nil
		}
	}
	return nil, fmt.Errorf("document not found: %s", id)
}

func (ks *KnowledgeStore) Search(query string, limit int) []Document {
	if limit <= 0 {
		limit = 5
	}
	queryWords := extractWords(query)
	scores := make(map[int]int)
	for _, word := range queryWords {
		if indices, ok := ks.index[word]; ok {
			for _, idx := range indices {
				scores[idx]++
			}
		}
	}
	type scoredDoc struct {
		idx   int
		score int
		doc   Document
	}
	var results []scoredDoc
	for idx, score := range scores {
		results = append(results, scoredDoc{idx: idx, score: score, doc: ks.documents[idx]})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].doc.UpdatedAt.After(results[j].doc.UpdatedAt)
		}
		return results[i].score > results[j].score
	})
	var docs []Document
	for i := 0; i < len(results) && i < limit; i++ {
		docs = append(docs, results[i].doc)
	}
	return docs
}

func (ks *KnowledgeStore) List() []Document {
	sort.Slice(ks.documents, func(i, j int) bool {
		return ks.documents[i].UpdatedAt.After(ks.documents[j].UpdatedAt)
	})
	result := make([]Document, len(ks.documents))
	copy(result, ks.documents)
	return result
}

func (ks *KnowledgeStore) Count() int {
	return len(ks.documents)
}
