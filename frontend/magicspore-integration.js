/**
 * MagicSpore.com PayPal Integration
 * Add this to your magicspore.com checkout page
 */

// PayPal Worker Integration
const PayPalWorker = {
    workerUrl: 'https://pay.magicspore.com',
    
    // Redirect to PayPal via worker
    redirectToPayPal: function(orderId) {
        if (!orderId) {
            console.error('Order ID is required for PayPal redirect');
            return;
        }
        
        // Show loading state
        this.showLoadingState();
        
        // Redirect to worker
        window.location.href = `${this.workerUrl}/redirect?orderId=${orderId}`;
    },
    
    // Add PayPal button to checkout
    addPayPalButton: function() {
        const checkoutButton = document.querySelector('.wc-proceed-to-checkout');
        if (!checkoutButton) return;
        
        // Create PayPal button
        const paypalButton = document.createElement('a');
        paypalButton.className = 'paypal-checkout-button';
        paypalButton.innerHTML = `
            <img src="https://www.paypalobjects.com/webstatic/mktg/Logo/pp-logo-200px.png" alt="PayPal" style="height: 40px;">
            <span>Pay with PayPal</span>
        `;
        
        // Add click handler
        paypalButton.addEventListener('click', (e) => {
            e.preventDefault();
            
            // Get current order ID (you may need to adjust this based on your setup)
            const orderId = this.getCurrentOrderId();
            if (orderId) {
                this.redirectToPayPal(orderId);
            } else {
                alert('Please complete your order details first.');
            }
        });
        
        // Add to checkout area
        checkoutButton.appendChild(paypalButton);
    },
    
    // Get current order ID (adjust based on your implementation)
    getCurrentOrderId: function() {
        // Method 1: From URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        let orderId = urlParams.get('order_id');
        
        // Method 2: From form data
        if (!orderId) {
            const orderInput = document.querySelector('input[name="order_id"]');
            orderId = orderInput ? orderInput.value : null;
        }
        
        // Method 3: From checkout form (WooCommerce specific)
        if (!orderId && typeof wc_checkout_params !== 'undefined') {
            orderId = wc_checkout_params.order_id;
        }
        
        // Method 4: From cart/session (requires AJAX call)
        if (!orderId) {
            // You might need to make an AJAX call to get the current order ID
            // This depends on your specific implementation
        }
        
        return orderId;
    },
    
    // Show loading state
    showLoadingState: function() {
        const button = document.querySelector('.paypal-checkout-button');
        if (button) {
            button.innerHTML = '<div class="loading-spinner"></div> Redirecting to PayPal...';
            button.style.pointerEvents = 'none';
        }
    },
    
    // Handle return from PayPal
    handlePayPalReturn: function() {
        const urlParams = new URLSearchParams(window.location.search);
        const payment = urlParams.get('payment');
        const orderId = urlParams.get('order');
        
        if (payment === 'success' || payment === 'confirmed') {
            // Show success message
            this.showSuccessMessage(orderId);
            
            // Redirect to thank you page after delay
            setTimeout(() => {
                window.location.href = '/dziekujemy?order=' + orderId;
            }, 3000);
        } else if (payment === 'cancelled') {
            this.showCancelMessage();
        }
    },
    
    // Show success message
    showSuccessMessage: function(orderId) {
        const message = document.createElement('div');
        message.className = 'paypal-success-message';
        message.innerHTML = `
            <div style="background: #d4edda; color: #155724; padding: 1rem; border-radius: 8px; margin: 1rem 0;">
                <h3>✅ Payment Successful!</h3>
                <p>Your PayPal payment has been processed successfully.</p>
                <p>Order #${orderId} is now being prepared.</p>
                <p>Redirecting to confirmation page...</p>
            </div>
        `;
        
        document.body.insertBefore(message, document.body.firstChild);
    },
    
    // Show cancel message
    showCancelMessage: function() {
        const message = document.createElement('div');
        message.className = 'paypal-cancel-message';
        message.innerHTML = `
            <div style="background: #f8d7da; color: #721c24; padding: 1rem; border-radius: 8px; margin: 1rem 0;">
                <h3>❌ Payment Cancelled</h3>
                <p>Your PayPal payment was cancelled.</p>
                <p>You can try again or choose a different payment method.</p>
            </div>
        `;
        
        document.body.insertBefore(message, document.body.firstChild);
    },
    
    // Initialize
    init: function() {
        // Add PayPal button when DOM is ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => {
                this.addPayPalButton();
                this.handlePayPalReturn();
            });
        } else {
            this.addPayPalButton();
            this.handlePayPalReturn();
        }
    }
};

// Auto-initialize
PayPalWorker.init();

// CSS Styles for PayPal button
const paypalStyles = `
<style>
.paypal-checkout-button {
    display: flex !important;
    align-items: center;
    gap: 10px;
    background: #0070ba;
    color: white !important;
    padding: 12px 24px;
    border-radius: 8px;
    text-decoration: none !important;
    font-weight: 600;
    font-size: 16px;
    margin-top: 15px;
    transition: all 0.3s ease;
    justify-content: center;
    cursor: pointer;
}

.paypal-checkout-button:hover {
    background: #005a9a !important;
    transform: translateY(-2px);
    box-shadow: 0 4px 15px rgba(0, 112, 186, 0.4);
}

.paypal-checkout-button img {
    height: 30px;
}

.loading-spinner {
    width: 20px;
    height: 20px;
    border: 2px solid #ffffff;
    border-top: 2px solid transparent;
    border-radius: 50%;
    animation: spin 1s linear infinite;
}

@keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
}

.paypal-success-message,
.paypal-cancel-message {
    position: fixed;
    top: 20px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 9999;
    max-width: 500px;
    width: 90%;
}
</style>
`;

// Inject styles
document.head.insertAdjacentHTML('beforeend', paypalStyles);