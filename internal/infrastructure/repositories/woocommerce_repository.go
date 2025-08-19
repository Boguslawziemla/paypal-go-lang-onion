package repositories

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"strings"
	"strconv"
	"time"
)

// WooCommerceRepository implements WooCommerce API operations
type WooCommerceRepository struct {
	magicConfig  WooCommerceConfig
	oitamConfig  WooCommerceConfig
	httpClient   *http.Client
	logger       interfaces.Logger
}

// WooCommerceConfig holds WooCommerce site configuration
type WooCommerceConfig struct {
	URL            string
	ConsumerKey    string
	ConsumerSecret string
	Timeout        time.Duration
	RetryAttempts  int
}

// NewWooCommerceRepository creates a new WooCommerce repository
func NewWooCommerceRepository(
	magicConfig, oitamConfig WooCommerceConfig,
	logger interfaces.Logger,
) interfaces.WooCommerceRepository {
	// Create HTTP client with timeouts and security settings
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Always verify SSL in production
			},
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	return &WooCommerceRepository{
		magicConfig: magicConfig,
		oitamConfig: oitamConfig,
		httpClient:  httpClient,
		logger:      logger,
	}
}

// GetMagicOrder fetches an order from MagicSpore site
func (r *WooCommerceRepository) GetMagicOrder(ctx context.Context, orderID string) (*entities.Order, error) {
	r.logger.Info("Fetching order from MagicSpore", map[string]interface{}{
		"order_id": orderID,
		"site":     "magicspore",
	})

	// Build API URL
	apiURL := fmt.Sprintf("%s/wp-json/wc/v3/orders/%s", 
		strings.TrimRight(r.magicConfig.URL, "/"), 
		orderID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	r.addWooCommerceAuth(req, r.magicConfig)
	r.addStandardHeaders(req)

	// Execute request with retry logic
	var wcOrder WooCommerceOrder
	err = r.executeWithRetry(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("order %s not found", orderID)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response
		if err := json.NewDecoder(resp.Body).Decode(&wcOrder); err != nil {
			return fmt.Errorf("failed to decode order response: %w", err)
		}

		return nil
	}, r.magicConfig.RetryAttempts)

	if err != nil {
		r.logger.Error("Failed to fetch MagicSpore order", err, map[string]interface{}{
			"order_id": orderID,
			"url":      apiURL,
		})
		return nil, err
	}

	// Convert to domain entity
	order, err := r.convertWooCommerceToEntity(&wcOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to convert order: %w", err)
	}

	r.logger.Info("Successfully fetched MagicSpore order", map[string]interface{}{
		"order_id": orderID,
		"status":   order.Status,
		"total":    order.Total.Amount,
		"currency": order.Currency,
	})

	return order, nil
}

// CreateOITAMOrder creates an order on OITAM site
func (r *WooCommerceRepository) CreateOITAMOrder(ctx context.Context, order *entities.Order) (*entities.Order, error) {
	r.logger.Info("Creating order on OITAM", map[string]interface{}{
		"original_order_number": order.Number,
		"total":                 order.Total.Amount,
		"currency":              order.Currency,
		"line_items":            len(order.LineItems),
	})

	// Convert order to OITAM format
	oitamOrderData := r.convertToOITAMOrder(order)

	// Build API URL
	apiURL := fmt.Sprintf("%s/wp-json/wc/v3/orders", 
		strings.TrimRight(r.oitamConfig.URL, "/"))

	// Serialize order data
	jsonData, err := json.Marshal(oitamOrderData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order data: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication and headers
	r.addWooCommerceAuth(req, r.oitamConfig)
	r.addStandardHeaders(req)

	// Execute request with retry logic
	var createdOrder WooCommerceOrder
	err = r.executeWithRetry(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			r.logger.Error("OITAM order creation failed", nil, map[string]interface{}{
				"status_code": resp.StatusCode,
				"response":    string(body),
				"request":     string(jsonData),
			})
			return fmt.Errorf("failed to create order, status: %d, response: %s", resp.StatusCode, string(body))
		}

		// Parse response
		if err := json.NewDecoder(resp.Body).Decode(&createdOrder); err != nil {
			return fmt.Errorf("failed to decode created order response: %w", err)
		}

		return nil
	}, r.oitamConfig.RetryAttempts)

	if err != nil {
		r.logger.Error("Failed to create OITAM order", err, map[string]interface{}{
			"original_order_number": order.Number,
			"url":                   apiURL,
		})
		return nil, err
	}

	// Convert to domain entity
	createdOrderEntity, err := r.convertWooCommerceToEntity(&createdOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to convert created order: %w", err)
	}

	r.logger.Info("Successfully created OITAM order", map[string]interface{}{
		"original_order_number": order.Number,
		"proxy_order_id":        createdOrderEntity.ID,
		"proxy_order_key":       createdOrderEntity.OrderKey,
		"status":                createdOrderEntity.Status,
	})

	return createdOrderEntity, nil
}

// GetOITAMOrder fetches an order from OITAM site
func (r *WooCommerceRepository) GetOITAMOrder(ctx context.Context, orderID string) (*entities.Order, error) {
	r.logger.Info("Fetching order from OITAM", map[string]interface{}{
		"order_id": orderID,
		"site":     "oitam",
	})

	// Build API URL
	apiURL := fmt.Sprintf("%s/wp-json/wc/v3/orders/%s", 
		strings.TrimRight(r.oitamConfig.URL, "/"), 
		orderID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	r.addWooCommerceAuth(req, r.oitamConfig)
	r.addStandardHeaders(req)

	// Execute request
	var wcOrder WooCommerceOrder
	err = r.executeWithRetry(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("order %s not found", orderID)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Parse response
		if err := json.NewDecoder(resp.Body).Decode(&wcOrder); err != nil {
			return fmt.Errorf("failed to decode order response: %w", err)
		}

		return nil
	}, r.oitamConfig.RetryAttempts)

	if err != nil {
		r.logger.Error("Failed to fetch OITAM order", err, map[string]interface{}{
			"order_id": orderID,
			"url":      apiURL,
		})
		return nil, err
	}

	// Convert to domain entity
	order, err := r.convertWooCommerceToEntity(&wcOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to convert order: %w", err)
	}

	r.logger.Info("Successfully fetched OITAM order", map[string]interface{}{
		"order_id": orderID,
		"status":   order.Status,
		"total":    order.Total.Amount,
	})

	return order, nil
}

// UpdateMagicOrderStatus updates order status on MagicSpore site
func (r *WooCommerceRepository) UpdateMagicOrderStatus(ctx context.Context, orderID string, status entities.OrderStatus) error {
	r.logger.Info("Updating MagicSpore order status", map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	})

	return r.updateOrderStatus(ctx, r.magicConfig, orderID, status)
}

// UpdateOITAMOrderStatus updates order status on OITAM site
func (r *WooCommerceRepository) UpdateOITAMOrderStatus(ctx context.Context, orderID string, status entities.OrderStatus) error {
	r.logger.Info("Updating OITAM order status", map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	})

	return r.updateOrderStatus(ctx, r.oitamConfig, orderID, status)
}

// UpdateMagicOrder updates an order on MagicSpore
func (r *WooCommerceRepository) UpdateMagicOrder(ctx context.Context, orderID string, order *entities.Order) error {
	r.logger.Info("Updating order on MagicSpore", map[string]interface{}{
		"order_id": orderID,
	})

	wcOrder := r.convertEntityToWooCommerce(order)
	return r.updateOrderFull(ctx, r.magicConfig, orderID, wcOrder)
}

// UpdateOITAMOrder updates an order on OITAM
func (r *WooCommerceRepository) UpdateOITAMOrder(ctx context.Context, orderID string, order *entities.Order) error {
	r.logger.Info("Updating order on OITAM", map[string]interface{}{
		"order_id": orderID,
	})

	wcOrder := r.convertEntityToWooCommerce(order)
	return r.updateOrderFull(ctx, r.oitamConfig, orderID, wcOrder)
}

// UpdateMagicOrderPayment updates payment information on MagicSpore order
func (r *WooCommerceRepository) UpdateMagicOrderPayment(ctx context.Context, orderID string, payment *entities.Payment) error {
	r.logger.Info("Updating MagicSpore order payment", map[string]interface{}{
		"order_id":       orderID,
		"payment_id":     payment.ID,
		"transaction_id": payment.TransactionID,
		"status":         payment.Status,
	})

	// Build update data
	updateData := map[string]interface{}{
		"payment_method":       "paypal",
		"payment_method_title": "PayPal",
		"transaction_id":       payment.TransactionID,
		"meta_data": []map[string]interface{}{
			{
				"key":   "_paypal_payment_id",
				"value": payment.PaymentID,
			},
			{
				"key":   "_payment_completed_at",
				"value": time.Now().Unix(),
			},
			{
				"key":   "_proxy_payment_processed",
				"value": "true",
			},
		},
	}

	// If payment is completed, update status
	if payment.IsCompleted() {
		updateData["status"] = "processing"
		updateData["date_paid"] = time.Now().Format("2006-01-02T15:04:05")
	}

	return r.updateOrder(ctx, r.magicConfig, orderID, updateData)
}

// Helper methods

// addWooCommerceAuth adds WooCommerce API authentication to request
func (r *WooCommerceRepository) addWooCommerceAuth(req *http.Request, config WooCommerceConfig) {
	// Use Basic Auth with consumer key and secret
	auth := base64.StdEncoding.EncodeToString(
		[]byte(config.ConsumerKey + ":" + config.ConsumerSecret),
	)
	req.Header.Set("Authorization", "Basic "+auth)
}

// addStandardHeaders adds standard headers to request
func (r *WooCommerceRepository) addStandardHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "PayPal-Proxy-Go/1.0")
}

// executeWithRetry executes HTTP request with retry logic
func (r *WooCommerceRepository) executeWithRetry(ctx context.Context, req *http.Request, handler func(*http.Response) error, maxRetries int) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}

			r.logger.Debug("Retrying API request", map[string]interface{}{
				"attempt": attempt,
				"url":     req.URL.String(),
			})
		}

		// Clone request for retry
		reqClone := req.Clone(ctx)

		resp, err := r.httpClient.Do(reqClone)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			if attempt == maxRetries {
				break
			}
			continue
		}

		// Handle response
		err = handler(resp)
		resp.Body.Close()

		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't retry on certain errors
		if resp.StatusCode == http.StatusNotFound || 
		   resp.StatusCode == http.StatusUnauthorized ||
		   resp.StatusCode == http.StatusForbidden {
			break
		}
	}

	return lastErr
}

// updateOrderStatus updates order status
func (r *WooCommerceRepository) updateOrderStatus(ctx context.Context, config WooCommerceConfig, orderID string, status entities.OrderStatus) error {
	updateData := map[string]interface{}{
		"status": string(status),
	}

	return r.updateOrder(ctx, config, orderID, updateData)
}

// updateOrder updates order with given data
func (r *WooCommerceRepository) updateOrder(ctx context.Context, config WooCommerceConfig, orderID string, updateData map[string]interface{}) error {
	// Build API URL
	apiURL := fmt.Sprintf("%s/wp-json/wc/v3/orders/%s", 
		strings.TrimRight(config.URL, "/"), 
		orderID)

	// Serialize update data
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	r.addWooCommerceAuth(req, config)
	r.addStandardHeaders(req)

	// Execute request
	return r.executeWithRetry(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to update order, status: %d, response: %s", resp.StatusCode, string(body))
		}
		return nil
	}, config.RetryAttempts)
}

// updateOrderFull updates order with complete data
func (r *WooCommerceRepository) updateOrderFull(ctx context.Context, config WooCommerceConfig, orderID string, orderData interface{}) error {
	jsonData, err := json.Marshal(orderData)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	url := fmt.Sprintf("%s/wp-json/wc/v3/orders/%s", config.URL, orderID)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	r.addWooCommerceAuth(req, config)
	r.addStandardHeaders(req)

	return r.executeWithRetry(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to update order, status: %d", resp.StatusCode)
		}
		return nil
	}, config.RetryAttempts)
}

// Data conversion methods

// WooCommerceOrder represents WooCommerce API order format
type WooCommerceOrder struct {
	ID              int                    `json:"id"`
	Number          string                 `json:"number"`
	Status          string                 `json:"status"`
	Currency        string                 `json:"currency"`
	Total           string                 `json:"total"`
	DateCreated     string                 `json:"date_created"`
	DateModified    string                 `json:"date_modified"`
	DatePaid        *string                `json:"date_paid"`
	PaymentMethod   string                 `json:"payment_method"`
	PaymentMethodTitle string              `json:"payment_method_title"`
	TransactionID   string                 `json:"transaction_id"`
	OrderKey        string                 `json:"order_key"`
	CustomerNote    string                 `json:"customer_note"`
	Billing         WooCommerceAddress     `json:"billing"`
	Shipping        WooCommerceAddress     `json:"shipping"`
	LineItems       []WooCommerceLineItem  `json:"line_items"`
	ShippingLines   []WooCommerceShipping  `json:"shipping_lines"`
	FeeLines        []WooCommerceFee       `json:"fee_lines"`
	TaxLines        []WooCommerceTax       `json:"tax_lines"`
	CouponLines     []WooCommerceCoupon    `json:"coupon_lines"`
	MetaData        []WooCommerceMetaData  `json:"meta_data"`
}

type WooCommerceAddress struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Company   string `json:"company"`
	Address1  string `json:"address_1"`
	Address2  string `json:"address_2"`
	City      string `json:"city"`
	State     string `json:"state"`
	Postcode  string `json:"postcode"`
	Country   string `json:"country"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

type WooCommerceLineItem struct {
	ID          int                   `json:"id"`
	Name        string                `json:"name"`
	ProductID   int                   `json:"product_id"`
	VariationID int                   `json:"variation_id"`
	Quantity    int                   `json:"quantity"`
	SKU         string                `json:"sku"`
	Price       string                `json:"price"`
	Subtotal    string                `json:"subtotal"`
	Total       string                `json:"total"`
	MetaData    []WooCommerceMetaData `json:"meta_data"`
}

type WooCommerceShipping struct {
	ID          int                   `json:"id"`
	MethodID    string                `json:"method_id"`
	MethodTitle string                `json:"method_title"`
	Total       string                `json:"total"`
	MetaData    []WooCommerceMetaData `json:"meta_data"`
}

type WooCommerceFee struct {
	ID       int                   `json:"id"`
	Name     string                `json:"name"`
	Total    string                `json:"total"`
	MetaData []WooCommerceMetaData `json:"meta_data"`
}

type WooCommerceTax struct {
	ID               int                   `json:"id"`
	RateCode         string                `json:"rate_code"`
	RateID           int                   `json:"rate_id"`
	Label            string                `json:"label"`
	Compound         bool                  `json:"compound"`
	TaxTotal         string                `json:"tax_total"`
	ShippingTaxTotal string                `json:"shipping_tax_total"`
	MetaData         []WooCommerceMetaData `json:"meta_data"`
}

type WooCommerceCoupon struct {
	ID          int                   `json:"id"`
	Code        string                `json:"code"`
	Discount    string                `json:"discount"`
	DiscountTax string                `json:"discount_tax"`
	MetaData    []WooCommerceMetaData `json:"meta_data"`
}

type WooCommerceMetaData struct {
	ID    int         `json:"id"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// convertWooCommerceToEntity converts WooCommerce API response to domain entity
func (r *WooCommerceRepository) convertWooCommerceToEntity(wcOrder *WooCommerceOrder) (*entities.Order, error) {
	totalAmount, err := strconv.ParseFloat(wcOrder.Total, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid total amount: %s", wcOrder.Total)
	}

	createdAt, err := time.Parse(time.RFC3339, wcOrder.DateCreated)
	if err != nil {
		createdAt = time.Now()
	}

	order := &entities.Order{
		ID:            wcOrder.ID,
		Number:        wcOrder.Number,
		Status:        entities.OrderStatus(wcOrder.Status),
		Currency:      wcOrder.Currency,
		Total:         entities.Money{Amount: totalAmount, Currency: wcOrder.Currency},
		PaymentMethod: wcOrder.PaymentMethod,
		PaymentMethodTitle: wcOrder.PaymentMethodTitle,
		TransactionID: wcOrder.TransactionID,
		DateCreated:   createdAt,
		OrderKey:      wcOrder.OrderKey,
		CustomerNote:  wcOrder.CustomerNote,
	}

	// Convert addresses
	order.Billing = r.convertAddress(wcOrder.Billing)
	order.Shipping = r.convertAddress(wcOrder.Shipping)

	// Convert line items
	for _, item := range wcOrder.LineItems {
		convertedItem, err := r.convertLineItem(item, wcOrder.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to convert line item: %w", err)
		}
		order.LineItems = append(order.LineItems, *convertedItem)
	}

	// Convert other lines
	for _, shipping := range wcOrder.ShippingLines {
		convertedShipping, err := r.convertShippingLine(shipping, wcOrder.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to convert shipping line: %w", err)
		}
		order.ShippingLines = append(order.ShippingLines, *convertedShipping)
	}

	return order, nil
}

// convertEntityToWooCommerce converts domain entity to WooCommerce API format
func (r *WooCommerceRepository) convertEntityToWooCommerce(order *entities.Order) map[string]interface{} {
	wcOrder := map[string]interface{}{
		"status":              string(order.Status),
		"currency":            order.Currency,
		"payment_method":      order.PaymentMethod,
		"payment_method_title": order.PaymentMethodTitle,
		"customer_note":       order.CustomerNote,
		"billing":             r.convertAddressToWC(order.Billing),
		"shipping":            r.convertAddressToWC(order.Shipping),
	}

	// Add line items
	var lineItems []map[string]interface{}
	for _, item := range order.LineItems {
		lineItems = append(lineItems, r.convertLineItemToWC(item))
	}
	wcOrder["line_items"] = lineItems

	// Add metadata for proxy orders
	if order.Number != "" {
		wcOrder["meta_data"] = []map[string]interface{}{
			{
				"key":   "_original_order_number",
				"value": order.Number,
			},
			{
				"key":   "_proxy_order",
				"value": "true",
			},
		}
	}

	return wcOrder
}

// convertToOITAMOrder converts order to OITAM format for creation
func (r *WooCommerceRepository) convertToOITAMOrder(order *entities.Order) map[string]interface{} {
	// Convert line items with anonymized names
	var lineItems []map[string]interface{}
	for i, item := range order.LineItems {
		lineItem := map[string]interface{}{
			"name":     fmt.Sprintf("Item %d", i+1), // Anonymized name
			"quantity": item.Quantity,
			"total":    item.Total.ToWooCommerceFormat(),
			"sku":      item.SKU, // Keep SKU for inventory tracking
			"meta_data": []map[string]interface{}{
				{
					"key":   "_original_name",
					"value": item.Name,
				},
				{
					"key":   "_original_product_id",
					"value": item.ProductID,
				},
			},
		}
		lineItems = append(lineItems, lineItem)
	}

	return map[string]interface{}{
		"status":   "pending",
		"currency": order.Currency,
		"total":    order.Total.ToWooCommerceFormat(),
		"billing":  r.convertAddressToWC(order.Billing),
		"shipping": r.convertAddressToWC(order.Shipping),
		"line_items":     lineItems,
		"payment_method": "paypal",
		"payment_method_title": "PayPal",
		"meta_data": []map[string]interface{}{
			{
				"key":   "_original_order_number",
				"value": order.Number,
			},
			{
				"key":   "_proxy_order",
				"value": "true",
			},
			{
				"key":   "_proxy_created_at",
				"value": time.Now().Unix(),
			},
		},
	}
}

// Address conversion helpers
func (r *WooCommerceRepository) convertAddress(wcAddr WooCommerceAddress) entities.Address {
	return entities.Address{
		FirstName: wcAddr.FirstName,
		LastName:  wcAddr.LastName,
		Company:   wcAddr.Company,
		Address1:  wcAddr.Address1,
		Address2:  wcAddr.Address2,
		City:      wcAddr.City,
		State:     wcAddr.State,
		Postcode:  wcAddr.Postcode,
		Country:   wcAddr.Country,
		Email:     wcAddr.Email,
		Phone:     wcAddr.Phone,
	}
}

func (r *WooCommerceRepository) convertAddressToWC(addr entities.Address) map[string]interface{} {
	return map[string]interface{}{
		"first_name": addr.FirstName,
		"last_name":  addr.LastName,
		"company":    addr.Company,
		"address_1":  addr.Address1,
		"address_2":  addr.Address2,
		"city":       addr.City,
		"state":      addr.State,
		"postcode":   addr.Postcode,
		"country":    addr.Country,
		"email":      addr.Email,
		"phone":      addr.Phone,
	}
}

// Line item conversion helpers
func (r *WooCommerceRepository) convertLineItem(wcItem WooCommerceLineItem, currency string) (*entities.LineItem, error) {
	priceAmount, err := strconv.ParseFloat(wcItem.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price: %s", wcItem.Price)
	}

	subtotalAmount, err := strconv.ParseFloat(wcItem.Subtotal, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid subtotal: %s", wcItem.Subtotal)
	}

	totalAmount, err := strconv.ParseFloat(wcItem.Total, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid total: %s", wcItem.Total)
	}

	return &entities.LineItem{
		ID:          wcItem.ID,
		Name:        wcItem.Name,
		ProductID:   wcItem.ProductID,
		VariationID: wcItem.VariationID,
		Quantity:    wcItem.Quantity,
		SKU:         wcItem.SKU,
		Price:       entities.Money{Amount: priceAmount, Currency: currency},
		Subtotal:    entities.Money{Amount: subtotalAmount, Currency: currency},
		Total:       entities.Money{Amount: totalAmount, Currency: currency},
	}, nil
}

func (r *WooCommerceRepository) convertLineItemToWC(item entities.LineItem) map[string]interface{} {
	return map[string]interface{}{
		"name":         item.Name,
		"product_id":   item.ProductID,
		"variation_id": item.VariationID,
		"quantity":     item.Quantity,
		"sku":          item.SKU,
	}
}

// Shipping line conversion helpers
func (r *WooCommerceRepository) convertShippingLine(wcShipping WooCommerceShipping, currency string) (*entities.ShippingLine, error) {
	totalAmount, err := strconv.ParseFloat(wcShipping.Total, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid shipping total: %s", wcShipping.Total)
	}

	return &entities.ShippingLine{
		ID:          wcShipping.ID,
		MethodID:    wcShipping.MethodID,
		MethodTitle: wcShipping.MethodTitle,
		Total:       entities.Money{Amount: totalAmount, Currency: currency},
	}, nil
}