package appwrite

import (
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/cryptx/cryptx-cli/internal/models"
)

// ── School Hackathon ──────────────────────────────────────────────────────────

// ListSchoolHackathon returns a paginated list of school hackathon registrations.
// search: exact document ID OR team name contains filter; empty = no filter.
func (s *Services) ListSchoolHackathon(page int, search string) ([]*models.SchoolHackathonRegistration, int, error) {
	collID := s.cfg.SchoolHackathonCollectionID
	if collID == "" {
		return nil, 0, fmt.Errorf("APPWRITE_SCHOOL_HACKATHON_COLLECTION_ID is not configured")
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
		return nil, 0, fmt.Errorf("list school hackathon registrations: %w", err)
	}

	regs := make([]*models.SchoolHackathonRegistration, 0, len(result.Documents))
	var rawList struct {
		Documents []json.RawMessage `json:"documents"`
	}
	if err := result.Decode(&rawList); err != nil {
		return nil, 0, fmt.Errorf("decode school hackathon document list: %w", err)
	}
	for _, raw := range rawList.Documents {
		var r models.SchoolHackathonRegistration
		if err := json.Unmarshal(raw, &r); err != nil {
			continue
		}
		regs = append(regs, &r)
	}
	return regs, result.Total, nil
}

// GetSchoolHackathon fetches a single school hackathon registration.
func (s *Services) GetSchoolHackathon(docID string) (*models.SchoolHackathonRegistration, error) {
	collID := s.cfg.SchoolHackathonCollectionID
	doc, err := s.DB.GetDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return nil, fmt.Errorf("get school hackathon registration %s: %w", docID, err)
	}
	var r models.SchoolHackathonRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode school hackathon registration: %w", err)
	}
	return &r, nil
}

// UpdateSchoolHackathon updates arbitrary fields on a school hackathon document.
func (s *Services) UpdateSchoolHackathon(docID string, data map[string]interface{}) (*models.SchoolHackathonRegistration, error) {
	collID := s.cfg.SchoolHackathonCollectionID
	doc, err := s.DB.UpdateDocument(
		s.cfg.DatabaseID, collID, docID,
		s.DB.WithUpdateDocumentData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("update school hackathon registration %s: %w", docID, err)
	}
	var r models.SchoolHackathonRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode updated school hackathon registration: %w", err)
	}
	return &r, nil
}

// ConfirmSchoolHackathon sets paymentStatus=verified on a school hackathon document.
func (s *Services) ConfirmSchoolHackathon(docID string) (*models.SchoolHackathonRegistration, error) {
	return s.UpdateSchoolHackathon(docID, map[string]interface{}{
		"paymentStatus": string(models.PaymentVerified),
	})
}

// DeleteSchoolHackathon removes a school hackathon registration document.
func (s *Services) DeleteSchoolHackathon(docID string) error {
	collID := s.cfg.SchoolHackathonCollectionID
	_, err := s.DB.DeleteDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return fmt.Errorf("delete school hackathon registration %s: %w", docID, err)
	}
	return nil
}

// DownloadSchoolHackathonLogo downloads a school hackathon team logo.
func (s *Services) DownloadSchoolHackathonLogo(fileID string) ([]byte, error) {
	bucketID := s.cfg.HackathonSchBucketID
	if bucketID == "" {
		return nil, fmt.Errorf("APPWRITE_HACKATHON_SCHOOL_BUCKET_ID is not configured")
	}
	data, err := s.Storage.GetFileDownload(bucketID, fileID)
	if err != nil {
		return nil, fmt.Errorf("download school hackathon logo %s: %w", fileID, err)
	}
	return *data, nil
}

// ── University Hackathon ──────────────────────────────────────────────────────

// ListUniversityHackathon returns a paginated list of university hackathon registrations.
// search: exact document ID OR team name contains filter; empty = no filter.
func (s *Services) ListUniversityHackathon(page int, search string) ([]*models.UniversityHackathonRegistration, int, error) {
	collID := s.cfg.UniversityHackathonCollectionID
	if collID == "" {
		return nil, 0, fmt.Errorf("APPWRITE_UNIVERSITY_HACKATHON_COLLECTION_ID is not configured")
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
		return nil, 0, fmt.Errorf("list university hackathon registrations: %w", err)
	}

	regs := make([]*models.UniversityHackathonRegistration, 0, len(result.Documents))
	var rawList struct {
		Documents []json.RawMessage `json:"documents"`
	}
	if err := result.Decode(&rawList); err != nil {
		return nil, 0, fmt.Errorf("decode university hackathon document list: %w", err)
	}
	for _, raw := range rawList.Documents {
		var r models.UniversityHackathonRegistration
		if err := json.Unmarshal(raw, &r); err != nil {
			continue
		}
		regs = append(regs, &r)
	}
	return regs, result.Total, nil
}

// GetUniversityHackathon fetches a single university hackathon registration.
func (s *Services) GetUniversityHackathon(docID string) (*models.UniversityHackathonRegistration, error) {
	collID := s.cfg.UniversityHackathonCollectionID
	doc, err := s.DB.GetDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return nil, fmt.Errorf("get university hackathon registration %s: %w", docID, err)
	}
	var r models.UniversityHackathonRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode university hackathon registration: %w", err)
	}
	return &r, nil
}

// DeleteUniversityHackathon removes a university hackathon registration document.
func (s *Services) DeleteUniversityHackathon(docID string) error {
	collID := s.cfg.UniversityHackathonCollectionID
	_, err := s.DB.DeleteDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return fmt.Errorf("delete university hackathon registration %s: %w", docID, err)
	}
	return nil
}

// DownloadUniversityHackathonLogo downloads a university hackathon team logo.
func (s *Services) DownloadUniversityHackathonLogo(fileID string) ([]byte, error) {
	bucketID := s.cfg.HackathonUniBucketID
	if bucketID == "" {
		return nil, fmt.Errorf("APPWRITE_HACKATHON_UNIVERSITY_BUCKET_ID is not configured")
	}
	data, err := s.Storage.GetFileDownload(bucketID, fileID)
	if err != nil {
		return nil, fmt.Errorf("download university hackathon logo %s: %w", fileID, err)
	}
	return *data, nil
}
