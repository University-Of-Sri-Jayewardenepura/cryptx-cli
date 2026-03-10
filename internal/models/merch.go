package models

import "time"

// MerchPaymentStatus represents the full payment lifecycle for a merch order.
type MerchPaymentStatus string

const (
	// Pre-order flow
	MerchStatusPendingPreOrder    MerchPaymentStatus = "pending_preorder_verification"
	MerchStatusPreOrderVerified   MerchPaymentStatus = "preorder_verified"
	MerchStatusPreOrderRejected   MerchPaymentStatus = "preorder_rejected"

	// Full payment flow
	MerchStatusPendingFullPayment     MerchPaymentStatus = "pending_full_payment_verification"
	MerchStatusFullyPaid              MerchPaymentStatus = "fully_paid"
	MerchStatusFullPaymentRejected    MerchPaymentStatus = "full_payment_rejected"

	// Completion / admin states
	MerchStatusDispatched MerchPaymentStatus = "dispatched"
	MerchStatusCancelled  MerchPaymentStatus = "cancelled"
)

// MerchOrder mirrors the merch-orders Appwrite collection.
type MerchOrder struct {
	// Appwrite document meta
	ID        string `json:"$id"`
	CreatedAt string `json:"$createdAt"`
	UpdatedAt string `json:"$updatedAt"`

	// Contact info
	FullName string `json:"fullName"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`

	// Product
	Product     string `json:"product"`
	ProductName string `json:"productName"`
	Size        string `json:"size"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int    `json:"unitPrice"`
	TotalPrice  int    `json:"totalPrice"`

	// Shipping
	StreetAddress string `json:"streetAddress"`
	City          string `json:"city"`
	District      string `json:"district"`
	PostalCode    string `json:"postalCode"`

	// Delivery & payment
	DeliveryMethod string `json:"deliveryMethod"` // "Event Day Collection" | "Courier"
	PaymentOption  string `json:"paymentOption"`  // "full" | "pre-order"

	// Payment slip
	PaymentSlipFileId string `json:"paymentSlipFileId,omitempty"`
	PaymentSlipUrl    string `json:"paymentSlipUrl,omitempty"`

	// System
	PaymentStatus MerchPaymentStatus `json:"paymentStatus"`
	SubmittedAt   string             `json:"submittedAt"`
}

// IsPendingVerification returns true when either a pre-order or full-payment
// slip is awaiting operator review.
func (o *MerchOrder) IsPendingVerification() bool {
	return o.PaymentStatus == MerchStatusPendingPreOrder ||
		o.PaymentStatus == MerchStatusPendingFullPayment
}

// CanBeDispatched returns true when the order is fully paid and not yet dispatched.
func (o *MerchOrder) CanBeDispatched() bool {
	return o.PaymentStatus == MerchStatusFullyPaid
}

// DisplayName returns a short human-readable name for list views.
func (o *MerchOrder) DisplayName() string {
	return o.FullName + " — " + o.ProductName
}

// PrimaryEmail returns the contact email.
func (o *MerchOrder) PrimaryEmail() string { return o.Email }

// CreatedAtTime parses the Appwrite ISO timestamp.
func (o *MerchOrder) CreatedAtTime() time.Time {
	t, _ := time.Parse(time.RFC3339, o.CreatedAt)
	return t
}
