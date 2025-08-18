package repositories

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"paypal-proxy/internal/domain/entities"
	"paypal-proxy/internal/domain/interfaces"
	"strconv"
	"time"
)

// WooCommerceRepository implements the WooCommerceRepository interface
type WooCommerceRepository struct {
	httpClient *http.Client
	config     interfaces.ConfigService
	logger     interfaces.Logger
}

// NewWooCommerceRepository creates a new WooCommerce repository
func NewWooCommerceRepository(config interfaces.ConfigService, logger interfaces.Logger) interfaces.WooCommerceRepository {
	return &WooCommerceRepository{
		httpClient: &http.Client{
			Timeout: time.Duration(config.GetServerConfig().APITimeout) * time.Second,
		},
		config: config,
		logger: logger,
	}
}

// GetMagicOrder retrieves an order from MagicSpore
func (r *WooCommerceRepository) GetMagicOrder(ctx context.Context, orderID string) (*entities.Order, error) {
	magicConfig := r.config.GetMagicSporeConfig()
	url := fmt.Sprintf("%s/%s", magicConfig.APIURL, orderID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", magicConfig.ConsumerKey, magicConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		r.logger.Error("Failed to get MagicSpore order", fmt.Errorf("API error"), map[string]interface{}{
			"order_id":    orderID,
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var orderData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orderData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	order := r.mapToOrder(orderData)
	
	r.logger.Info("Successfully retrieved MagicSpore order", map[string]interface{}{
		"order_id":     orderID,
		"order_number": order.Number,
		"status":       order.Status,
		"total":        order.Total.Amount,
	})
	
	return order, nil
}

// CreateOITAMOrder creates a new order on OITAM
func (r *WooCommerceRepository) CreateOITAMOrder(ctx context.Context, order *entities.Order) (*entities.Order, error) {
	oitamConfig := r.config.GetOITAMConfig()
	
	orderData := r.mapFromOrder(order)
	jsonData, err := json.Marshal(orderData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", oitamConfig.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", oitamConfig.ConsumerKey, oitamConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		r.logger.Error("Failed to create OITAM order", fmt.Errorf("API error"), map[string]interface{}{
			"order_number": order.Number,
			"status_code":  resp.StatusCode,
			"response":     string(body),
		})
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var createdOrderData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createdOrderData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	createdOrder := r.mapToOrder(createdOrderData)
	
	r.logger.Info("Successfully created OITAM order", map[string]interface{}{
		"oitam_order_id": createdOrder.ID,
		"order_number":   createdOrder.Number,
		"total":          createdOrder.Total.Amount,
	})
	
	return createdOrder, nil
}

// GetOITAMOrder retrieves an order from OITAM
func (r *WooCommerceRepository) GetOITAMOrder(ctx context.Context, orderID string) (*entities.Order, error) {
	oitamConfig := r.config.GetOITAMConfig()
	url := fmt.Sprintf("%s/%s", oitamConfig.APIURL, orderID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", oitamConfig.ConsumerKey, oitamConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var orderData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orderData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return r.mapToOrder(orderData), nil
}

// UpdateMagicOrder updates an order on MagicSpore
func (r *WooCommerceRepository) UpdateMagicOrder(ctx context.Context, orderID string, order *entities.Order) error {
	magicConfig := r.config.GetMagicSporeConfig()
	url := fmt.Sprintf("%s/%s", magicConfig.APIURL, orderID)
	
	updateData := r.mapFromOrder(order)
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", magicConfig.ConsumerKey, magicConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		r.logger.Error("Failed to update MagicSpore order", fmt.Errorf("API error"), map[string]interface{}{
			"order_id":    orderID,
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully updated MagicSpore order", map[string]interface{}{
		"order_id": orderID,
		"status":   order.Status,
	})

	return nil
}

// UpdateMagicOrderStatus updates only the status of a MagicSpore order
func (r *WooCommerceRepository) UpdateMagicOrderStatus(ctx context.Context, orderID string, status entities.OrderStatus) error {
	magicConfig := r.config.GetMagicSporeConfig()
	url := fmt.Sprintf("%s/%s", magicConfig.APIURL, orderID)
	
	updateData := map[string]interface{}{
		"status": string(status),
	}
	
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal update data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", magicConfig.ConsumerKey, magicConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		r.logger.Error("Failed to update MagicSpore order status", fmt.Errorf("API error"), map[string]interface{}{
			"order_id":    orderID,
			"status":      status,
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully updated MagicSpore order status", map[string]interface{}{
		"order_id": orderID,
		"status":   status,
	})

	return nil
}

// UpdateMagicOrderPayment updates a MagicSpore order with payment information
func (r *WooCommerceRepository) UpdateMagicOrderPayment(ctx context.Context, orderID string, payment *entities.Payment) error {
	updateData := map[string]interface{}{
		"status":              string(entities.StatusProcessing),
		"payment_method":      string(payment.Method),
		"payment_method_title": "PayPal",
		"transaction_id":      payment.TransactionID,
		"date_paid":          payment.ProcessedAt.Format(time.RFC3339),
		"meta_data": []map[string]interface{}{
			{
				"key":   "_paypal_payment_id",
				"value": payment.PaymentID,
			},
			{
				"key":   "_paypal_payer_id",
				"value": payment.PayerID,
			},
		},
	}

	magicConfig := r.config.GetMagicSporeConfig()
	url := fmt.Sprintf("%s/%s", magicConfig.APIURL, orderID)
	
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal payment data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", magicConfig.ConsumerKey, magicConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		r.logger.Error("Failed to update MagicSpore order with payment", fmt.Errorf("API error"), map[string]interface{}{
			"order_id":    orderID,
			"payment_id":  payment.PaymentID,
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully updated MagicSpore order with payment", map[string]interface{}{
		"order_id":       orderID,
		"payment_id":     payment.PaymentID,
		"transaction_id": payment.TransactionID,
	})

	return nil
}

// UpdateOITAMOrder updates an order on OITAM
func (r *WooCommerceRepository) UpdateOITAMOrder(ctx context.Context, orderID string, order *entities.Order) error {
	oitamConfig := r.config.GetOITAMConfig()
	url := fmt.Sprintf("%s/%s", oitamConfig.APIURL, orderID)
	
	updateData := r.mapFromOrder(order)
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth
	auth := base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s:%s", oitamConfig.ConsumerKey, oitamConfig.ConsumerSecret),
	))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// mapToOrder maps WooCommerce API data to domain Order entity
func (r *WooCommerceRepository) mapToOrder(data map[string]interface{}) *entities.Order {
	order := &entities.Order{}
	
	if id, ok := data["id"].(float64); ok {
		order.ID = int(id)
	}
	
	if number, ok := data["number"].(string); ok {
		order.Number = number
	}
	
	if status, ok := data["status"].(string); ok {
		order.Status = entities.OrderStatus(status)
	}
	
	if currency, ok := data["currency"].(string); ok {
		order.Currency = currency
	}
	
	if total, ok := data["total"].(string); ok {
		if totalFloat, err := strconv.ParseFloat(total, 64); err == nil {
			order.Total = entities.Money{
				Amount:   totalFloat,
				Currency: order.Currency,
			}
		}
	}
	
	if paymentMethod, ok := data["payment_method"].(string); ok {
		order.PaymentMethod = paymentMethod
	}
	
	if paymentMethodTitle, ok := data["payment_method_title"].(string); ok {
		order.PaymentMethodTitle = paymentMethodTitle
	}
	
	if transactionID, ok := data["transaction_id"].(string); ok {
		order.TransactionID = transactionID
	}
	
	if orderKey, ok := data["order_key"].(string); ok {
		order.OrderKey = orderKey
	}
	
	// Parse dates
	if dateCreated, ok := data["date_created"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, dateCreated); err == nil {
			order.DateCreated = parsed
		}
	}
	
	if datePaid, ok := data["date_paid"].(string); ok && datePaid != "" {
		if parsed, err := time.Parse(time.RFC3339, datePaid); err == nil {
			order.DatePaid = &parsed
		}
	}
	
	// Map billing and shipping addresses
	if billing, ok := data["billing"].(map[string]interface{}); ok {
		order.Billing = r.mapToAddress(billing)
	}
	
	if shipping, ok := data["shipping"].(map[string]interface{}); ok {
		order.Shipping = r.mapToAddress(shipping)
	}
	
	// Map line items
	if lineItems, ok := data["line_items"].([]interface{}); ok {
		for _, item := range lineItems {
			if itemMap, ok := item.(map[string]interface{}); ok {
				order.LineItems = append(order.LineItems, r.mapToLineItem(itemMap, order.Currency))
			}
		}
	}
	
	return order
}

// mapToAddress maps address data to domain Address entity
func (r *WooCommerceRepository) mapToAddress(data map[string]interface{}) entities.Address {
	address := entities.Address{}
	
	if firstName, ok := data["first_name"].(string); ok {
		address.FirstName = firstName
	}
	if lastName, ok := data["last_name"].(string); ok {
		address.LastName = lastName
	}
	if company, ok := data["company"].(string); ok {
		address.Company = company
	}
	if address1, ok := data["address_1"].(string); ok {
		address.Address1 = address1
	}
	if address2, ok := data["address_2"].(string); ok {
		address.Address2 = address2
	}
	if city, ok := data["city"].(string); ok {
		address.City = city
	}
	if state, ok := data["state"].(string); ok {
		address.State = state
	}
	if postcode, ok := data["postcode"].(string); ok {
		address.Postcode = postcode
	}
	if country, ok := data["country"].(string); ok {
		address.Country = country
	}
	if email, ok := data["email"].(string); ok {
		address.Email = email
	}
	if phone, ok := data["phone"].(string); ok {
		address.Phone = phone
	}
	
	return address
}

// mapToLineItem maps line item data to domain LineItem entity
func (r *WooCommerceRepository) mapToLineItem(data map[string]interface{}, currency string) entities.LineItem {
	item := entities.LineItem{}
	
	if id, ok := data["id"].(float64); ok {
		item.ID = int(id)
	}
	if name, ok := data["name"].(string); ok {
		item.Name = name
	}
	if productId, ok := data["product_id"].(float64); ok {
		item.ProductID = int(productId)
	}
	if variationId, ok := data["variation_id"].(float64); ok {
		item.VariationID = int(variationId)
	}
	if quantity, ok := data["quantity"].(float64); ok {
		item.Quantity = int(quantity)
	}
	if sku, ok := data["sku"].(string); ok {
		item.SKU = sku
	}
	
	// Parse prices
	if price, ok := data["price"].(float64); ok {
		item.Price = entities.Money{Amount: price, Currency: currency}
	}
	if subtotal, ok := data["subtotal"].(string); ok {
		if subtotalFloat, err := strconv.ParseFloat(subtotal, 64); err == nil {
			item.Subtotal = entities.Money{Amount: subtotalFloat, Currency: currency}
		}
	}
	if total, ok := data["total"].(string); ok {
		if totalFloat, err := strconv.ParseFloat(total, 64); err == nil {
			item.Total = entities.Money{Amount: totalFloat, Currency: currency}
		}
	}
	
	return item
}

// mapFromOrder maps domain Order entity to WooCommerce API data
func (r *WooCommerceRepository) mapFromOrder(order *entities.Order) map[string]interface{} {
	data := map[string]interface{}{
		"number":               order.Number,
		"status":               string(order.Status),
		"currency":             order.Currency,
		"payment_method":       order.PaymentMethod,
		"payment_method_title": order.PaymentMethodTitle,
		"set_paid":             false,
		"customer_id":          0, // Guest checkout
		"total":                fmt.Sprintf("%.2f", order.Total.Amount),
		"billing":              r.mapFromAddress(order.Billing),
		"shipping":             r.mapFromAddress(order.Shipping),
		"line_items":           r.mapFromLineItems(order.LineItems),
		"customer_note":        order.CustomerNote,
	}
	
	// Add metadata if present
	if len(order.MetaData) > 0 {
		metaData := make([]map[string]interface{}, len(order.MetaData))
		for i, meta := range order.MetaData {
			metaData[i] = map[string]interface{}{
				"key":   meta.Key,
				"value": meta.Value,
			}
		}
		data["meta_data"] = metaData
	}
	
	return data
}

// mapFromAddress maps domain Address entity to WooCommerce API data
func (r *WooCommerceRepository) mapFromAddress(address entities.Address) map[string]interface{} {
	return map[string]interface{}{
		"first_name": address.FirstName,
		"last_name":  address.LastName,
		"company":    address.Company,
		"address_1":  address.Address1,
		"address_2":  address.Address2,
		"city":       address.City,
		"state":      address.State,
		"postcode":   address.Postcode,
		"country":    address.Country,
		"email":      address.Email,
		"phone":      address.Phone,
	}
}

// mapFromLineItems maps domain LineItem entities to WooCommerce API data
func (r *WooCommerceRepository) mapFromLineItems(items []entities.LineItem) []map[string]interface{} {
	lineItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		lineItems[i] = map[string]interface{}{
			"name":         item.Name,
			"product_id":   item.ProductID,
			"variation_id": item.VariationID,
			"quantity":     item.Quantity,
			"sku":          item.SKU,
			"price":        item.Price.Amount,
			"subtotal":     fmt.Sprintf("%.2f", item.Subtotal.Amount),
			"total":        fmt.Sprintf("%.2f", item.Total.Amount),
		}
	}
	return lineItems
}