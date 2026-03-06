package models

import "time"

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
	DeliveryMethod string `json:"deliveryMethod"` // "University Pickup" | "Courier"
	PaymentOption  string `json:"paymentOption"`  // "full" | "pre-order"

	// Payment slip
	PaymentSlipFileId string `json:"paymentSlipFileId,omitempty"`
	PaymentSlipUrl    string `json:"paymentSlipUrl,omitempty"`

	// System
	PaymentStatus PaymentStatus `json:"paymentStatus"`
	SubmittedAt   string        `json:"submittedAt"`
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
