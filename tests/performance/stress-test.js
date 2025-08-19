import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Stress test configuration - higher load to find breaking point
export const options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '3m', target: 200 },  // Ramp up to 200 users
    { duration: '5m', target: 300 },  // Ramp up to 300 users (stress level)
    { duration: '5m', target: 500 },  // Ramp up to 500 users (breaking point)
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000'], // 95% of requests under 5s (more lenient for stress)
    http_req_failed: ['rate<0.3'],     // Allow higher error rate during stress
    errors: ['rate<0.3'],              // Custom error rate threshold
  },
};

const BASE_URL = __ENV.TEST_BASE_URL || 'http://localhost:8080';

export default function () {
  // Mix of different request types with varying complexity
  const testType = Math.random();
  
  if (testType < 0.4) {
    // 40% - Simple health checks
    testHealthCheck();
  } else if (testType < 0.7) {
    // 30% - Payment redirects (more complex)
    testPaymentRedirect();
  } else if (testType < 0.9) {
    // 20% - API calls
    testAPIEndpoints();
  } else {
    // 10% - Webhook simulation (most complex)
    testWebhookSimulation();
  }
  
  // Shorter sleep during stress test
  sleep(Math.random() * 2);
}

function testHealthCheck() {
  const response = http.get(`${BASE_URL}/health`);
  
  const success = check(response, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 3000ms': (r) => r.timings.duration < 3000,
  });

  if (!success) {
    errorRate.add(1);
  }
}

function testPaymentRedirect() {
  const orderId = `STRESS_${Math.floor(Math.random() * 10000)}`;
  const response = http.get(`${BASE_URL}/redirect?orderId=${orderId}`, {
    redirects: 0,
    timeout: '10s',
  });
  
  const success = check(response, {
    'redirect status is 302 or 400': (r) => r.status === 302 || r.status === 400,
    'redirect response time < 5000ms': (r) => r.timings.duration < 5000,
  });

  if (!success) {
    errorRate.add(1);
  }
}

function testAPIEndpoints() {
  const endpoints = [
    '/api/v1/health',
    '/ping',
    '/ready',
    '/live',
  ];
  
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const response = http.get(`${BASE_URL}${endpoint}`, {
    timeout: '10s',
  });
  
  const success = check(response, {
    'API endpoint responds': (r) => r.status >= 200 && r.status < 500,
    'API response time < 5000ms': (r) => r.timings.duration < 5000,
  });

  if (!success) {
    errorRate.add(1);
  }
}

function testWebhookSimulation() {
  const webhookData = {
    id: `stress_webhook_${Math.floor(Math.random() * 10000)}`,
    event_type: 'PAYMENT.CAPTURE.COMPLETED',
    resource: {
      id: `payment_${Math.floor(Math.random() * 10000)}`,
      custom_id: `order_${Math.floor(Math.random() * 10000)}`,
      amount: {
        value: (Math.random() * 100 + 10).toFixed(2),
        currency_code: 'USD'
      }
    }
  };
  
  const response = http.post(
    `${BASE_URL}/webhook`,
    JSON.stringify(webhookData),
    {
      headers: {
        'Content-Type': 'application/json',
      },
      timeout: '15s',
    }
  );
  
  const success = check(response, {
    'webhook responds': (r) => r.status >= 200 && r.status < 500,
    'webhook response time < 10000ms': (r) => r.timings.duration < 10000,
  });

  if (!success) {
    errorRate.add(1);
  }
}

// Test concurrent connections to a single endpoint
export function testConcurrentConnections() {
  const promises = [];
  
  // Fire 10 concurrent requests
  for (let i = 0; i < 10; i++) {
    promises.push(
      http.asyncRequest('GET', `${BASE_URL}/health`, null, {
        timeout: '30s',
      })
    );
  }
  
  const responses = Promise.all(promises);
  
  check(responses, {
    'all concurrent requests completed': (responses) => responses.length === 10,
  });
}

// Test memory and resource exhaustion scenarios
export function testResourceExhaustion() {
  // Create large payload to test memory handling
  const largePayload = 'x'.repeat(1024 * 100); // 100KB payload
  
  const response = http.post(
    `${BASE_URL}/webhook`,
    largePayload,
    {
      headers: {
        'Content-Type': 'text/plain',
      },
      timeout: '30s',
    }
  );
  
  check(response, {
    'large payload handled gracefully': (r) => r.status >= 400 && r.status < 500,
  });
}

export function handleSummary(data) {
  const results = {
    summary: {
      test_type: 'Stress Test',
      duration_seconds: Math.round(data.state.testRunDurationMs / 1000),
      max_vus: data.metrics.vus_max.values.max,
      total_requests: data.metrics.http_reqs.values.count,
      request_rate: Math.round(data.metrics.http_reqs.values.rate * 100) / 100,
      avg_response_time: Math.round(data.metrics.http_req_duration.values.avg),
      p95_response_time: Math.round(data.metrics.http_req_duration.values['p(95)']),
      error_rate: Math.round(data.metrics.http_req_failed.values.rate * 10000) / 100,
    },
    thresholds: {},
    recommendations: []
  };
  
  // Extract threshold results
  Object.entries(data.metrics).forEach(([name, metric]) => {
    if (metric.thresholds) {
      Object.entries(metric.thresholds).forEach(([threshold, result]) => {
        results.thresholds[`${name}: ${threshold}`] = result.ok ? 'PASS' : 'FAIL';
      });
    }
  });
  
  // Generate recommendations based on results
  if (results.summary.error_rate > 20) {
    results.recommendations.push('High error rate detected. Consider increasing server resources or implementing better error handling.');
  }
  
  if (results.summary.p95_response_time > 3000) {
    results.recommendations.push('High response times detected. Consider optimizing database queries and adding caching.');
  }
  
  if (results.summary.request_rate < 50) {
    results.recommendations.push('Low request processing rate. Check for bottlenecks in the application.');
  }
  
  const htmlReport = `
<!DOCTYPE html>
<html>
<head>
    <title>PayPal Proxy Stress Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { background-color: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 5px; margin-bottom: 30px; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 20px 0; }
        .metric { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); color: white; padding: 20px; border-radius: 10px; text-align: center; }
        .metric h3 { margin: 0 0 10px 0; font-size: 14px; text-transform: uppercase; }
        .metric .value { font-size: 28px; font-weight: bold; }
        .metric .unit { font-size: 14px; opacity: 0.8; }
        .recommendations { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 20px; border-radius: 5px; margin: 20px 0; }
        .recommendations h3 { color: #856404; margin: 0 0 15px 0; }
        .recommendations ul { margin: 0; padding-left: 20px; }
        .recommendations li { margin: 5px 0; color: #856404; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; background-color: white; }
        th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
        th { background-color: #f8f9fa; font-weight: 600; }
        .pass { color: #28a745; font-weight: bold; }
        .fail { color: #dc3545; font-weight: bold; }
        .warning { color: #ffc107; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 PayPal Proxy Stress Test Report</h1>
            <p><strong>Generated:</strong> ${new Date().toISOString()}</p>
            <p><strong>Test Type:</strong> High-Load Stress Testing</p>
        </div>
        
        <div class="metrics">
            <div class="metric">
                <h3>Duration</h3>
                <div class="value">${results.summary.duration_seconds}</div>
                <div class="unit">seconds</div>
            </div>
            <div class="metric">
                <h3>Max Users</h3>
                <div class="value">${results.summary.max_vus}</div>
                <div class="unit">concurrent</div>
            </div>
            <div class="metric">
                <h3>Total Requests</h3>
                <div class="value">${results.summary.total_requests.toLocaleString()}</div>
                <div class="unit">requests</div>
            </div>
            <div class="metric">
                <h3>Request Rate</h3>
                <div class="value">${results.summary.request_rate}</div>
                <div class="unit">req/sec</div>
            </div>
            <div class="metric">
                <h3>Avg Response</h3>
                <div class="value">${results.summary.avg_response_time}</div>
                <div class="unit">ms</div>
            </div>
            <div class="metric">
                <h3>95th Percentile</h3>
                <div class="value">${results.summary.p95_response_time}</div>
                <div class="unit">ms</div>
            </div>
            <div class="metric">
                <h3>Error Rate</h3>
                <div class="value">${results.summary.error_rate}%</div>
                <div class="unit">errors</div>
            </div>
        </div>
        
        ${results.recommendations.length > 0 ? `
        <div class="recommendations">
            <h3>📋 Recommendations</h3>
            <ul>
                ${results.recommendations.map(rec => `<li>${rec}</li>`).join('')}
            </ul>
        </div>
        ` : ''}
        
        <h2>🎯 Threshold Results</h2>
        <table>
            <thead>
                <tr><th>Metric & Threshold</th><th>Result</th></tr>
            </thead>
            <tbody>
                ${Object.entries(results.thresholds).map(([threshold, result]) => 
                  `<tr>
                    <td>${threshold}</td>
                    <td class="${result.toLowerCase()}">${result}</td>
                  </tr>`
                ).join('')}
            </tbody>
        </table>
        
        <div style="margin-top: 30px; padding: 20px; background-color: #e3f2fd; border-radius: 5px;">
            <h3 style="color: #1976d2; margin: 0 0 10px 0;">💡 About This Test</h3>
            <p style="margin: 0; color: #1976d2;">
                This stress test gradually increases load to find the breaking point of your PayPal Proxy service. 
                It helps identify performance bottlenecks and resource limits under extreme conditions.
            </p>
        </div>
    </div>
</body>
</html>`;

  return {
    'stress-test-summary.json': JSON.stringify(results, null, 2),
    'stress-test-summary.html': htmlReport,
  };
}