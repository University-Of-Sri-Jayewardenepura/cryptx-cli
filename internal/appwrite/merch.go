package appwrite

import (
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/query"
	"github.com/cryptx/cryptx-cli/internal/models"
)

// ListMerch returns a paginated list of merch orders.
// filter: "" (all) or any MerchPaymentStatus string.
// search: name/email contains search; empty = no filter.
func (s *Services) ListMerch(page int, filter string, search string) ([]*models.MerchOrder, int, error) {
	collID := s.cfg.MerchCollectionID
	if collID == "" {
		return nil, 0, fmt.Errorf("APPWRITE_MERCH_COLLECTION_ID is not configured")
	}

	queries := []string{
		query.Limit(pageSize),
		query.Offset(page * pageSize),
		query.OrderDesc("submittedAt"),
	}
	if filter != "" {
		queries = append(queries, query.Equal("paymentStatus", filter))
	}
	if search != "" {
		queries = append(queries, query.Or([]string{
			query.Equal("$id", search),
			query.Contains("fullName", search),
			query.Contains("email", search),
		}))
	}

	result, err := s.DB.ListDocuments(
		s.cfg.DatabaseID, collID,
		s.DB.WithListDocumentsQueries(queries),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list merch orders: %w", err)
	}

	orders := make([]*models.MerchOrder, 0, len(result.Documents))
	var rawList struct {
		Documents []json.RawMessage `json:"documents"`
	}
	if err := result.Decode(&rawList); err != nil {
		return nil, 0, fmt.Errorf("decode merch order list: %w", err)
	}
	for _, raw := range rawList.Documents {
		var o models.MerchOrder
		if err := json.Unmarshal(raw, &o); err != nil {
			continue
		}
		orders = append(orders, &o)
	}
	return orders, result.Total, nil
}

// GetMerch fetches a single merch order by document ID.
func (s *Services) GetMerch(docID string) (*models.MerchOrder, error) {
	collID := s.cfg.MerchCollectionID
	doc, err := s.DB.GetDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return nil, fmt.Errorf("get merch order %s: %w", docID, err)
	}
	var o models.MerchOrder
	if err := doc.Decode(&o); err != nil {
		return nil, fmt.Errorf("decode merch order: %w", err)
	}
	return &o, nil
}

// updateMerchStatus is the internal helper that sets paymentStatus on a merch doc.
func (s *Services) updateMerchStatus(docID string, status models.MerchPaymentStatus) (*models.MerchOrder, error) {
	collID := s.cfg.MerchCollectionID
	data := map[string]interface{}{
		"paymentStatus": string(status),
	}
	doc, err := s.DB.UpdateDocument(
		s.cfg.DatabaseID, collID, docID,
		s.DB.WithUpdateDocumentData(data),
	)
	if err != nil {
		return nil, fmt.Errorf("update merch order %s status to %s: %w", docID, status, err)
	}
	var o models.MerchOrder
	if err := doc.Decode(&o); err != nil {
		return nil, fmt.Errorf("decode updated merch order: %w", err)
	}
	return &o, nil
}

// ConfirmMerchPreOrder sets paymentStatus=preorder_verified.
func (s *Services) ConfirmMerchPreOrder(docID string) (*models.MerchOrder, error) {
	return s.updateMerchStatus(docID, models.MerchStatusPreOrderVerified)
}

// RejectMerchPreOrder sets paymentStatus=preorder_rejected.
func (s *Services) RejectMerchPreOrder(docID string) (*models.MerchOrder, error) {
	return s.updateMerchStatus(docID, models.MerchStatusPreOrderRejected)
}

// ConfirmMerchFullPayment sets paymentStatus=fully_paid.
func (s *Services) ConfirmMerchFullPayment(docID string) (*models.MerchOrder, error) {
	return s.updateMerchStatus(docID, models.MerchStatusFullyPaid)
}

// RejectMerchFullPayment sets paymentStatus=full_payment_rejected.
func (s *Services) RejectMerchFullPayment(docID string) (*models.MerchOrder, error) {
	return s.updateMerchStatus(docID, models.MerchStatusFullPaymentRejected)
}

// DispatchMerch sets paymentStatus=dispatched.
func (s *Services) DispatchMerch(docID string) (*models.MerchOrder, error) {
	return s.updateMerchStatus(docID, models.MerchStatusDispatched)
}

// CancelMerch sets paymentStatus=cancelled.
func (s *Services) CancelMerch(docID string) (*models.MerchOrder, error) {
	return s.updateMerchStatus(docID, models.MerchStatusCancelled)
}

// DeleteMerch removes a merch order document.
func (s *Services) DeleteMerch(docID string) error {
	collID := s.cfg.MerchCollectionID
	_, err := s.DB.DeleteDocument(s.cfg.DatabaseID, collID, docID)
	if err != nil {
		return fmt.Errorf("delete merch order %s: %w", docID, err)
	}
	return nil
}

// DownloadMerchPaymentSlip fetches the payment slip bytes from the merch bucket.
func (s *Services) DownloadMerchPaymentSlip(fileID string) ([]byte, error) {
	bucketID := s.cfg.MerchBucketID
	if bucketID == "" {
		return nil, fmt.Errorf("APPWRITE_MERCH_BUCKET_ID is not configured")
	}
	data, err := s.Storage.GetFileDownload(bucketID, fileID)
	if err != nil {
		return nil, fmt.Errorf("download merch payment slip %s: %w", fileID, err)
	}
	return *data, nil
}
