<?php
/**
 * PayPal Configuration for OITAM WooCommerce
 * Quick setup script for PayPal integration
 */

// PayPal Gateway Configuration
function setup_oitam_paypal_gateway() {
    // Enable PayPal Standard gateway
    update_option('woocommerce_paypal_settings', array(
        'enabled' => 'yes',
        'title' => 'PayPal',
        'description' => 'Pay securely with your PayPal account.',
        'email' => 'your-paypal-email@oitam.com', // CHANGE THIS
        'receiver_email' => 'your-paypal-email@oitam.com', // CHANGE THIS
        'identity_token' => '', // Optional
        'invoice_prefix' => 'OITAM-',
        'send_shipping' => 'yes',
        'address_override' => 'no',
        'paymentaction' => 'sale',
        'page_style' => '',
        'shipping' => 'yes',
        'testmode' => 'no', // Set to 'yes' for sandbox testing
        'debug' => 'yes',
        'sandbox_email' => 'sandbox@oitam.com', // For testing
        'form_submission_method' => 'yes',
        'api_username' => '', // For PayPal Pro (optional)
        'api_password' => '', // For PayPal Pro (optional)
        'api_signature' => '', // For PayPal Pro (optional)
    ));
    
    echo "✅ PayPal gateway configured successfully!\n";
}

// WooCommerce REST API Keys Setup
function setup_oitam_api_keys() {
    global $wpdb;
    
    // Create API key for the worker
    $key_data = array(
        'user_id' => 1, // Admin user
        'description' => 'Cloudflare Worker API Access',
        'permissions' => 'read_write',
        'consumer_key' => 'ck_' . wp_generate_password(40, false),
        'consumer_secret' => 'cs_' . wp_generate_password(40, false),
        'nonces' => '',
        'truncated_key' => substr('ck_' . wp_generate_password(40, false), -7)
    );
    
    $wpdb->insert(
        $wpdb->prefix . 'woocommerce_api_keys',
        $key_data
    );
    
    echo "🔑 API Keys generated:\n";
    echo "Consumer Key: " . $key_data['consumer_key'] . "\n";
    echo "Consumer Secret: " . $key_data['consumer_secret'] . "\n";
    echo "\n📋 Add these to your Cloudflare Worker environment variables:\n";
    echo "OITAM_CONSUMER_KEY=" . $key_data['consumer_key'] . "\n";
    echo "OITAM_CONSUMER_SECRET=" . $key_data['consumer_secret'] . "\n";
    
    return $key_data;
}

// Configure WooCommerce settings for proxy orders
function setup_oitam_woocommerce_settings() {
    // Allow guest checkout
    update_option('woocommerce_enable_guest_checkout', 'yes');
    
    // Simplify checkout
    update_option('woocommerce_enable_signup_and_login_from_checkout', 'no');
    update_option('woocommerce_enable_myaccount_registration', 'no');
    
    // Currency settings (adjust as needed)
    update_option('woocommerce_currency', 'USD');
    update_option('woocommerce_price_thousand_sep', ',');
    update_option('woocommerce_price_decimal_sep', '.');
    update_option('woocommerce_price_num_decimals', '2');
    
    // Disable reviews and ratings for proxy products
    update_option('woocommerce_enable_reviews', 'no');
    update_option('woocommerce_review_rating_verification_required', 'no');
    
    // Set default country
    update_option('woocommerce_default_country', 'US:*');
    
    echo "⚙️  WooCommerce settings configured for proxy orders!\n";
}

// Create sample products (matching magicspore.com structure)
function create_oitam_sample_products() {
    // Create sample categories
    $categories = array(
        'Mushroom Supplies' => 'Equipment and supplies for mushroom cultivation',
        'Grow Kits' => 'Ready-to-grow mushroom kits',
        'Substrates' => 'Growing substrates and materials'
    );
    
    foreach ($categories as $cat_name => $cat_desc) {
        wp_insert_term($cat_name, 'product_cat', array('description' => $cat_desc));
    }
    
    // Create sample products (these will be used for proxy orders)
    $products = array(
        array(
            'name' => 'Generic Item 1',
            'price' => '19.99',
            'sku' => 'GENERIC-001',
            'category' => 'Mushroom Supplies'
        ),
        array(
            'name' => 'Generic Item 2', 
            'price' => '29.99',
            'sku' => 'GENERIC-002',
            'category' => 'Grow Kits'
        ),
        array(
            'name' => 'Generic Item 3',
            'price' => '39.99', 
            'sku' => 'GENERIC-003',
            'category' => 'Substrates'
        )
    );
    
    foreach ($products as $product_data) {
        $product = new WC_Product_Simple();
        $product->set_name($product_data['name']);
        $product->set_regular_price($product_data['price']);
        $product->set_sku($product_data['sku']);
        $product->set_short_description('Generic product for payment processing');
        $product->set_description('This is a generic product used for payment processing purposes.');
        $product->set_status('publish');
        $product->set_virtual(true); // Virtual product
        $product->save();
        
        // Assign category
        $category = get_term_by('name', $product_data['category'], 'product_cat');
        if ($category) {
            wp_set_post_terms($product->get_id(), array($category->term_id), 'product_cat');
        }
    }
    
    echo "📦 Sample products created!\n";
}

// Main setup function
function run_oitam_setup() {
    echo "🚀 Setting up OITAM WooCommerce for PayPal proxy...\n\n";
    
    // Check if WooCommerce is active
    if (!class_exists('WooCommerce')) {
        echo "❌ Error: WooCommerce is not installed or activated!\n";
        return;
    }
    
    setup_oitam_woocommerce_settings();
    setup_oitam_paypal_gateway();
    $api_keys = setup_oitam_api_keys();
    create_oitam_sample_products();
    
    echo "\n✅ OITAM WooCommerce setup complete!\n";
    echo "\n📋 Next steps:\n";
    echo "1. Update PayPal email in PayPal settings\n";
    echo "2. Add API keys to Cloudflare Worker\n";
    echo "3. Test payment flow\n";
    echo "4. Switch PayPal to live mode when ready\n";
}

// Run setup if accessed directly
if (php_sapi_name() === 'cli') {
    run_oitam_setup();
}
?>