<?php
// OITAM WooCommerce Theme Functions
// Copy design from magicspore.com

// Enqueue styles and scripts
function oitam_enqueue_scripts() {
    wp_enqueue_style('oitam-style', get_stylesheet_uri());
    wp_enqueue_style('oitam-magicspore-style', get_template_directory_uri() . '/css/magicspore-clone.css');
    wp_enqueue_script('oitam-script', get_template_directory_uri() . '/js/oitam-custom.js', array('jquery'), '1.0.0', true);
}
add_action('wp_enqueue_scripts', 'oitam_enqueue_scripts');

// WooCommerce support
add_theme_support('woocommerce');
add_theme_support('wc-product-gallery-zoom');
add_theme_support('wc-product-gallery-lightbox');
add_theme_support('wc-product-gallery-slider');

// Remove WooCommerce default styles
add_filter('woocommerce_enqueue_styles', '__return_false');

// Custom PayPal return URL handling
function oitam_paypal_return_handler() {
    if (isset($_GET['return_from_paypal']) && $_GET['return_from_paypal'] === '1') {
        $order_id = isset($_GET['order_id']) ? sanitize_text_field($_GET['order_id']) : '';
        $status = isset($_GET['status']) ? sanitize_text_field($_GET['status']) : '';
        
        if ($order_id && $status === 'success') {
            // Update order status
            $order = wc_get_order($order_id);
            if ($order) {
                $order->update_status('processing', 'Payment completed via PayPal proxy');
                $order->payment_complete();
                
                // Redirect back to magicspore.com success page
                $magic_success_url = 'https://magicspore.com/dziekujemy?order=' . $order_id;
                wp_redirect($magic_success_url);
                exit;
            }
        }
    }
}
add_action('init', 'oitam_paypal_return_handler');

// Hide checkout fields (since we use proxy orders)
function oitam_hide_checkout_fields($fields) {
    // Keep minimal required fields
    $fields['billing']['billing_country']['required'] = true;
    
    // Hide unnecessary fields for proxy orders
    unset($fields['billing']['billing_first_name']);
    unset($fields['billing']['billing_last_name']);
    unset($fields['billing']['billing_address_1']);
    unset($fields['billing']['billing_city']);
    unset($fields['billing']['billing_postcode']);
    unset($fields['billing']['billing_phone']);
    unset($fields['billing']['billing_email']);
    
    return $fields;
}
add_filter('woocommerce_checkout_fields', 'oitam_hide_checkout_fields');

// Auto-populate checkout for proxy orders
function oitam_auto_populate_checkout() {
    if (is_wc_endpoint_url('order-pay')) {
        ?>
        <script>
        jQuery(document).ready(function($) {
            // Auto-fill proxy order data
            $('#billing_first_name').val('Customer');
            $('#billing_last_name').val('Order');
            $('#billing_address_1').val('Private');
            $('#billing_city').val('Private');
            $('#billing_postcode').val('00000');
            $('#billing_email').val('noreply@oitam.com');
            
            // Hide billing form for cleaner checkout
            $('.woocommerce-billing-fields').hide();
        });
        </script>
        <?php
    }
}
add_action('wp_footer', 'oitam_auto_populate_checkout');

// Custom order received page
function oitam_custom_thankyou_redirect($order_id) {
    $order = wc_get_order($order_id);
    if ($order) {
        // Check if this is a proxy order
        $is_proxy = $order->get_meta('_proxy_order');
        $original_order_id = $order->get_meta('_original_order_id');
        
        if ($is_proxy && $original_order_id) {
            // Redirect back to magicspore.com with success
            wp_redirect('https://magicspore.com/dziekujemy?order=' . $original_order_id . '&payment=success');
            exit;
        }
    }
}
add_action('woocommerce_thankyou', 'oitam_custom_thankyou_redirect');

// Enable REST API for guest orders
add_filter('woocommerce_rest_check_permissions', function($permission, $context, $object_id, $post_type) {
    if ($context === 'create' && $post_type === 'shop_order') {
        return true; // Allow guest order creation via API
    }
    return $permission;
}, 10, 4);

// Custom PayPal payment gateway settings
function oitam_paypal_gateway_settings() {
    return array(
        'enabled' => 'yes',
        'title' => 'PayPal',
        'description' => 'Pay securely with PayPal',
        'sandbox' => 'no', // Set to 'yes' for testing
        'email' => get_option('admin_email'), // Your PayPal email
    );
}

// Add custom CSS for magicspore.com clone
function oitam_magicspore_styles() {
    ?>
    <style>
    /* Clone magicspore.com design */
    body {
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        background: #f8f9fa;
    }
    
    .site-header {
        background: #2c3e50;
        color: white;
        padding: 1rem 0;
    }
    
    .site-title {
        font-size: 2rem;
        font-weight: bold;
        margin: 0;
    }
    
    .woocommerce-checkout {
        max-width: 800px;
        margin: 2rem auto;
        padding: 2rem;
        background: white;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    }
    
    .wc-proceed-to-checkout .checkout-button {
        background: #e67e22 !important;
        color: white !important;
        border: none;
        padding: 1rem 2rem;
        border-radius: 6px;
        font-size: 1.1rem;
        font-weight: 600;
    }
    
    .wc-proceed-to-checkout .checkout-button:hover {
        background: #d35400 !important;
    }
    
    /* Hide unnecessary elements for proxy checkout */
    .woocommerce-checkout .woocommerce-billing-fields h3,
    .woocommerce-checkout .woocommerce-additional-fields,
    .woocommerce-checkout .create-account {
        display: none !important;
    }
    
    /* PayPal button styling */
    .payment_method_paypal img {
        max-height: 40px;
    }
    
    #place_order {
        background: #0070ba !important;
        color: white !important;
        border: none;
        padding: 1rem 2rem;
        border-radius: 6px;
        font-size: 1.1rem;
        width: 100%;
    }
    
    #place_order:hover {
        background: #005a9a !important;
    }
    </style>
    <?php
}
add_action('wp_head', 'oitam_magicspore_styles');
?>