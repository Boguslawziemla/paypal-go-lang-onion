package entities

import (
	"fmt"
	"time"
)

// Order represents a WooCommerce order entity
type Order struct {
	ID                int
	Number            string
	Status            OrderStatus
	Currency          string
	Total             Money
	PaymentMethod     string
	PaymentMethodTitle string
	TransactionID     string
	DateCreated       time.Time
	DatePaid          *time.Time
	OrderKey          string
	CustomerNote      string
	Billing           Address
	Shipping          Address
	LineItems         []LineItem
	ShippingLines     []ShippingLine
	FeeLines          []FeeLine
	TaxLines          []TaxLine
	CouponLines       []CouponLine
	MetaData          []MetaData
}

// OrderStatus represents the status of an order
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusProcessing OrderStatus = "processing"
	StatusOnHold     OrderStatus = "on-hold"
	StatusCompleted  OrderStatus = "completed"
	StatusCancelled  OrderStatus = "cancelled"
	StatusRefunded   OrderStatus = "refunded"
	StatusFailed     OrderStatus = "failed"
)

// Money represents monetary value
type Money struct {
	Amount   float64
	Currency string
}

// Address represents billing or shipping address
type Address struct {
	FirstName string
	LastName  string
	Company   string
	Address1  string
	Address2  string
	City      string
	State     string
	Postcode  string
	Country   string
	Email     string
	Phone     string
}

// LineItem represents an order line item
type LineItem struct {
	ID          int
	Name        string
	ProductID   int
	VariationID int
	Quantity    int
	SKU         string
	Price       Money
	Subtotal    Money
	Total       Money
	MetaData    []MetaData
}

// ShippingLine represents shipping information
type ShippingLine struct {
	ID          int
	MethodID    string
	MethodTitle string
	Total       Money
	MetaData    []MetaData
}

// FeeLine represents additional fees
type FeeLine struct {
	ID       int
	Name     string
	Total    Money
	MetaData []MetaData
}

// TaxLine represents tax information
type TaxLine struct {
	ID               int
	RateCode         string
	RateID           int
	Label            string
	Compound         bool
	TaxTotal         Money
	ShippingTaxTotal Money
	MetaData         []MetaData
}

// CouponLine represents coupon/discount information
type CouponLine struct {
	ID          int
	Code        string
	Discount    Money
	DiscountTax Money
	MetaData    []MetaData
}

// MetaData represents additional metadata
type MetaData struct {
	ID    int
	Key   string
	Value interface{}
}

// IsPaymentCompleted checks if the order payment is completed
func (o *Order) IsPaymentCompleted() bool {
	completedStatuses := []OrderStatus{
		StatusCompleted,
		StatusProcessing,
		StatusOnHold,
	}
	
	for _, status := range completedStatuses {
		if o.Status == status {
			return true
		}
	}
	return false
}

// CanBeProcessed checks if the order can be processed for payment
func (o *Order) CanBeProcessed() bool {
	return o.Status == StatusPending && o.Total.Amount > 0
}

// ToAnonymousOrder creates an anonymized version of the order for proxy processing
func (o *Order) ToAnonymousOrder() *Order {
	anonymousOrder := &Order{
		Number:            o.Number, // Keep same order number
		Status:            StatusPending,
		Currency:          o.Currency,
		Total:             o.Total,
		PaymentMethod:     "paypal",
		PaymentMethodTitle: "PayPal",
		DateCreated:       time.Now(),
		CustomerNote:      "",
		Billing: Address{
			FirstName: "Customer",
			LastName:  "Order",
			Address1:  "Private",
			City:      "Private",
			Postcode:  "00000",
			Country:   o.Billing.Country, // Keep for PayPal requirements
			Email:     "noreply@oitam.com",
		},
		Shipping: Address{
			FirstName: "Customer",
			LastName:  "Order",
			Address1:  "Private",
			City:      "Private",
			Postcode:  "00000",
			Country:   o.Shipping.Country,
		},
		LineItems:     o.anonymizeLineItems(),
		ShippingLines: o.ShippingLines,
		FeeLines:      o.FeeLines,
		TaxLines:      o.TaxLines,
		CouponLines:   []CouponLine{}, // Remove coupon info
		MetaData: []MetaData{
			{Key: "_original_order_id", Value: o.ID},
			{Key: "_proxy_order", Value: "true"},
		},
	}
	
	return anonymousOrder
}

// anonymizeLineItems creates anonymous line items
func (o *Order) anonymizeLineItems() []LineItem {
	var anonymousItems []LineItem
	
	for i, item := range o.LineItems {
		anonymousItem := LineItem{
			Name:        formatGenericItemName(i + 1),
			ProductID:   0, // No product reference
			VariationID: 0,
			Quantity:    item.Quantity,
			SKU:         item.SKU, // Keep original SKU for inventory
			Price:       item.Price,
			Subtotal:    item.Subtotal,
			Total:       item.Total,
			MetaData:    []MetaData{}, // No product metadata
		}
		anonymousItems = append(anonymousItems, anonymousItem)
	}
	
	return anonymousItems
}

// formatGenericItemName creates generic item names
func formatGenericItemName(index int) string {
	return fmt.Sprintf("Item %d", index)
}