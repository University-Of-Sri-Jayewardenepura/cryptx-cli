package appwrite

import (
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/cryptx/cryptx-cli/internal/models"
)

// ListDesignathon returns a paginated list of designathon registrations.
// search: exact document ID OR team name contains filter; empty = no filter.
func (s *Services) ListDesignathon(page int, search string) ([]*models.DesignathonRegistration, int, error) {
	collID := s.cfg.DesignathonCollectionID
	if collID == "" {
		return nil, 0, fmt.Errorf("APPWRITE_DESIGNATHON_COLLECTION_ID is not configured")
	}

	queries := []string{
		query.Limit(pageSize),
		query.Offset(page * pageSize),
		query.OrderDesc("submittedAt"),
	}
	if search != "" {
		queries = append(queries, query.Or([]string{
			query.Equal("$id", search),
			query.Contains("teamName", search),
		}))
	}

	result, err := s.DB.ListDocuments(
		s.cfg.DatabaseID, collID,
		s.DB.WithListDocumentsQueries(queries),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list designathon registrations: %w", err)
	}

	regs := make([]*models.DesignathonRegistration, 0, len(result.Documents))
	var rawList struct {
		Documents []json.RawMessage `json:"documents"`
	}
	if err := result.Decode(&rawList); err != nil {
		return nil, 0, fmt.Errorf("decode designathon document list: %w", err)
	}
	for _, raw := range rawList.Documents {
		var r models.DesignathonRegistration
		if err := json.Unmarshal(raw, &r); err != nil {
			continue
		}
		regs = append(regs, &r)
	}
	return regs, result.Total, nil
}

// GetDesignathon fetches a single designathon registration by document ID.
func (s *Services) GetDesignathon(docID string) (*models.DesignathonRegistration, error) {
	collID := s.cfg.DesignathonCollectionID
	doc, err := s.DB.GetDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return nil, fmt.Errorf("get designathon registration %s: %w", docID, err)
	}
	var r models.DesignathonRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode designathon registration: %w", err)
	}
	return &r, nil
}

// UpdateDesignathon updates arbitrary fields on a designathon document.
func (s *Services) UpdateDesignathon(docID string, data map[string]interface{}) (*models.DesignathonRegistration, error) {
	collID := s.cfg.DesignathonCollectionID
	doc, err := s.DB.UpdateDocument(
		s.cfg.DatabaseID, collID, docID,
		s.DB.WithUpdateDocumentData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("update designathon registration %s: %w", docID, err)
	}
	var r models.DesignathonRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode updated designathon registration: %w", err)
	}
	return &r, nil
}

// DeleteDesignathon removes a designathon registration document.
func (s *Services) DeleteDesignathon(docID string) error {
	collID := s.cfg.DesignathonCollectionID
	_, err := s.DB.DeleteDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return fmt.Errorf("delete designathon registration %s: %w", docID, err)
	}
	return nil
}

// DownloadTeamLogo downloads the designathon team logo bytes from the storage bucket.
func (s *Services) DownloadTeamLogo(fileID string) ([]byte, error) {
	bucketID := s.cfg.DesignathonBucketID
	if bucketID == "" {
		return nil, fmt.Errorf("APPWRITE_DESIGNATHON_BUCKET_ID is not configured")
	}
	data, err := s.Storage.GetFileDownload(bucketID, fileID)
	if err != nil {
		return nil, fmt.Errorf("download team logo %s: %w", fileID, err)
	}
	return *data, nil
}
