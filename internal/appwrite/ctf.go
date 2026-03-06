package appwrite

import (
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/cryptx/cryptx-cli/internal/models"
)

const pageSize = 25

// ListCTF returns a paginated list of CTF registrations.
// filter: "" (all), "pending_verification", "verified", or "rejected".
// search: exact document ID OR team name contains filter; empty = no filter.
func (s *Services) ListCTF(page int, filter string, search string) ([]*models.CTFRegistration, int, error) {
	collID := s.cfg.CTFCollectionID
	if collID == "" {
		return nil, 0, fmt.Errorf("APPWRITE_CTF_COLLECTION_ID is not configured")
	}

	queries := []string{
		query.Limit(pageSize),
		query.Offset(page * pageSize),
		query.OrderDesc("submittedAt"),
	}
	switch filter {
	case string(models.PaymentPending), string(models.PaymentVerified), string(models.PaymentRejected):
		queries = append(queries, query.Equal("paymentStatus", filter))
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
		return nil, 0, fmt.Errorf("list CTF registrations: %w", err)
	}

	regs := make([]*models.CTFRegistration, 0, len(result.Documents))
	var rawList struct {
		Documents []json.RawMessage `json:"documents"`
	}
	if err := result.Decode(&rawList); err != nil {
		return nil, 0, fmt.Errorf("decode CTF document list: %w", err)
	}
	for _, raw := range rawList.Documents {
		var r models.CTFRegistration
		if err := json.Unmarshal(raw, &r); err != nil {
			continue
		}
		regs = append(regs, &r)
	}
	return regs, result.Total, nil
}

// GetCTF fetches a single CTF registration by document ID.
func (s *Services) GetCTF(docID string) (*models.CTFRegistration, error) {
	collID := s.cfg.CTFCollectionID
	doc, err := s.DB.GetDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return nil, fmt.Errorf("get CTF registration %s: %w", docID, err)
	}
	var r models.CTFRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode CTF registration: %w", err)
	}
	return &r, nil
}

// ConfirmCTF sets paymentStatus=verified on a CTF document.
func (s *Services) ConfirmCTF(docID string) (*models.CTFRegistration, error) {
	collID := s.cfg.CTFCollectionID
	data := map[string]interface{}{
		"paymentStatus": string(models.PaymentVerified),
	}
	doc, err := s.DB.UpdateDocument(
		s.cfg.DatabaseID, collID, docID,
		s.DB.WithUpdateDocumentData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("confirm CTF registration %s: %w", docID, err)
	}
	var r models.CTFRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode confirmed CTF registration: %w", err)
	}
	return &r, nil
}

// RejectCTF sets paymentStatus=rejected on a CTF document.
func (s *Services) RejectCTF(docID string) (*models.CTFRegistration, error) {
	collID := s.cfg.CTFCollectionID
	data := map[string]interface{}{
		"paymentStatus": string(models.PaymentRejected),
	}
	doc, err := s.DB.UpdateDocument(
		s.cfg.DatabaseID, collID, docID,
		s.DB.WithUpdateDocumentData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("reject CTF registration %s: %w", docID, err)
	}
	var r models.CTFRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode rejected CTF registration: %w", err)
	}
	return &r, nil
}

// UpdateCTF updates arbitrary fields on a CTF document.
func (s *Services) UpdateCTF(docID string, data map[string]interface{}) (*models.CTFRegistration, error) {
	collID := s.cfg.CTFCollectionID
	doc, err := s.DB.UpdateDocument(
		s.cfg.DatabaseID, collID, docID,
		s.DB.WithUpdateDocumentData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("update CTF registration %s: %w", docID, err)
	}
	var r models.CTFRegistration
	if err := doc.Decode(&r); err != nil {
		return nil, fmt.Errorf("decode updated CTF registration: %w", err)
	}
	return &r, nil
}

// DeleteCTF removes a CTF registration document.
func (s *Services) DeleteCTF(docID string) error {
	collID := s.cfg.CTFCollectionID
	_, err := s.DB.DeleteDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return fmt.Errorf("delete CTF registration %s: %w", docID, err)
	}
	return nil
}

// DownloadPaymentSlip downloads the CTF payment slip bytes from the storage bucket.
func (s *Services) DownloadPaymentSlip(fileID string) ([]byte, error) {
	bucketID := s.cfg.CTFBucketID
	if bucketID == "" {
		return nil, fmt.Errorf("APPWRITE_CTF_BUCKET_ID is not configured")
	}
	data, err := s.Storage.GetFileDownload(bucketID, fileID)
	if err != nil {
		return nil, fmt.Errorf("download payment slip %s: %w", fileID, err)
	}
	return *data, nil
}
