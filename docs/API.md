# 📚 PayPal Proxy API Documentation

## Base URL
```
http://localhost:8080
```

## Authentication
No authentication required for public endpoints. WooCommerce API authentication is handled internally.

## Endpoints

### Health Check
```http
GET /health
```

**Response:**
```json
{
    "status": "OK",
    "timestamp": "2024-01-01T12:00:00Z",
    "version": "1.0.0",
    "uptime": "1h23m45s"
}
```

### Payment Redirect (Main Endpoint)
```http
GET /redirect?orderId={order_id}
```

**Parameters:**
- `orderId` (required): Order ID from magicspore.com

**Response:**
- `302 Redirect` to PayPal checkout on oitam.com

**Example:**
```bash
curl -I "http://localhost:8080/redirect?orderId=123"
```

### PayPal Return (Success)
```http
GET /paypal-return?order_id={order_id}&oitam_order_id={oitam_id}&status=success&paymentId={payment_id}&PayerID={payer_id}
```

**Parameters:**
- `order_id` (required): Original MagicSpore order ID
- `oitam_order_id`: OITAM proxy order ID
- `status`: Payment status
- `paymentId`: PayPal payment ID
- `PayerID`: PayPal payer ID

**Response:**
- `302 Redirect` to success page on magicspore.com

### PayPal Cancel
```http
GET /paypal-cancel?order_id={order_id}&oitam_order_id={oitam_id}
```

**Parameters:**
- `order_id` (required): Original MagicSpore order ID
- `oitam_order_id`: OITAM proxy order ID

**Response:**
- `302 Redirect` to cancel page on magicspore.com

### Webhook Handler
```http
POST /webhook
```

**Request Body:**
```json
{
    "event_type": "PAYMENT.CAPTURE.COMPLETED",
    "resource": {
        "id": "payment_id",
        "custom_id": "order_id",
        "invoice_id": "order_id"
    }
}
```

**Response:**
```json
{
    "message": "Webhook processed successfully"
}
```

## API Routes (v1)

### Get Order
```http
GET /api/v1/order/{id}
```

**Response:**
```json
{
    "id": 123,
    "number": "123",
    "status": "processing",
    "currency": "USD",
    "total": "99.99",
    "payment_method": "paypal",
    "date_created": "2024-01-01T12:00:00Z",
    "billing": {...},
    "line_items": [...]
}
```

### Get Order Status
```http
GET /api/v1/status/{id}
```

**Response:**
```json
{
    "order_id": "123",
    "status": "processing",
    "total": "99.99",
    "currency": "USD",
    "payment_method": "paypal",
    "date_created": "2024-01-01T12:00:00Z",
    "date_paid": "2024-01-01T12:05:00Z"
}
```

## Error Responses

All errors return HTTP status codes with JSON error objects:

```json
{
    "error": "Error message",
    "message": "Detailed error description",
    "code": 400
}
```

### Common Error Codes
- `400` - Bad Request (missing parameters)
- `404` - Not Found (order doesn't exist)
- `500` - Internal Server Error (API or processing error)

## Rate Limiting

Default rate limits:
- 100 requests per minute per IP
- Configurable via `RATE_LIMIT` environment variable

## CORS

CORS is enabled for all origins. In production, configure specific origins for security.

## Security Headers

All responses include security headers:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `X-XSS-Protection: 1; mode=block`
- Content Security Policy headers

## Logging

All requests are logged with:
- IP address
- Request method and path
- Response status
- Response time
- User agent

Log levels: `debug`, `info`, `warn`, `error`