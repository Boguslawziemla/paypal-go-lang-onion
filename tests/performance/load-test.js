import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 users
    { duration: '5m', target: 10 },   // Stay at 10 users
    { duration: '2m', target: 25 },   // Ramp up to 25 users
    { duration: '5m', target: 25 },   // Stay at 25 users
    { duration: '2m', target: 50 },   // Ramp up to 50 users
    { duration: '5m', target: 50 },   // Stay at 50 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests must be under 1s
    http_req_failed: ['rate<0.1'],     // Error rate must be less than 10%
    errors: ['rate<0.1'],              // Custom error rate threshold
  },
};

const BASE_URL = __ENV.TEST_BASE_URL || 'http://localhost:8080';

export default function () {
  // Test 1: Health check endpoint
  testHealthCheck();
  sleep(1);

  // Test 2: Payment redirect flow
  testPaymentRedirect();
  sleep(1);

  // Test 3: API endpoints
  testAPIEndpoints();
  sleep(1);

  // Test 4: Error scenarios
  testErrorScenarios();
  sleep(2);
}

function testHealthCheck() {
  const response = http.get(`${BASE_URL}/health`);
  
  const success = check(response, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 500ms': (r) => r.timings.duration < 500,
    'health check has correct content-type': (r) => r.headers['Content-Type'].includes('application/json'),
    'health check has status field': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.hasOwnProperty('status');
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
  }
}

function testPaymentRedirect() {
  const orderIds = [
    'LOAD_TEST_001',
    'LOAD_TEST_002',
    'LOAD_TEST_003',
    'LOAD_TEST_004',
    'LOAD_TEST_005',
  ];
  
  const orderId = orderIds[Math.floor(Math.random() * orderIds.length)];
  const response = http.get(`${BASE_URL}/redirect?orderId=${orderId}`, {
    redirects: 0, // Don't follow redirects
  });
  
  const success = check(response, {
    'redirect status is 302': (r) => r.status === 302,
    'redirect response time < 2000ms': (r) => r.timings.duration < 2000,
    'redirect has Location header': (r) => r.headers.hasOwnProperty('Location'),
    'redirect Location contains oitam': (r) => {
      const location = r.headers['Location'];
      return location && location.includes('oitam');
    },
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
  const response = http.get(`${BASE_URL}${endpoint}`);
  
  const success = check(response, {
    'API endpoint status is 200': (r) => r.status === 200,
    'API response time < 1000ms': (r) => r.timings.duration < 1000,
    'API has correct content-type': (r) => r.headers['Content-Type'].includes('application/json'),
  });

  if (!success) {
    errorRate.add(1);
  }
}

function testErrorScenarios() {
  // Test invalid order ID
  const invalidResponse = http.get(`${BASE_URL}/redirect?orderId=invalid@#$%`);
  
  check(invalidResponse, {
    'invalid order ID returns 400': (r) => r.status === 400,
    'invalid order ID response time < 1000ms': (r) => r.timings.duration < 1000,
  });

  // Test missing order ID
  const missingResponse = http.get(`${BASE_URL}/redirect`);
  
  check(missingResponse, {
    'missing order ID returns 400': (r) => r.status === 400,
    'missing order ID response time < 1000ms': (r) => r.timings.duration < 1000,
  });

  // Test non-existent endpoint
  const notFoundResponse = http.get(`${BASE_URL}/non-existent-endpoint`);
  
  check(notFoundResponse, {
    'non-existent endpoint returns 404': (r) => r.status === 404,
    'non-existent endpoint response time < 1000ms': (r) => r.timings.duration < 1000,
  });
}

export function handleSummary(data) {
  return {
    'load-test-summary.json': JSON.stringify(data, null, 2),
    'load-test-summary.html': generateHTMLReport(data),
  };
}

function generateHTMLReport(data) {
  const date = new Date().toISOString();
  
  return `
<!DOCTYPE html>
<html>
<head>
    <title>PayPal Proxy Load Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .metrics { display: flex; flex-wrap: wrap; gap: 20px; margin: 20px 0; }
        .metric { background-color: #e8f4fd; padding: 15px; border-radius: 5px; min-width: 200px; }
        .metric h3 { margin: 0 0 10px 0; color: #2c5aa0; }
        .pass { color: green; }
        .fail { color: red; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>PayPal Proxy Load Test Report</h1>
        <p><strong>Generated:</strong> ${date}</p>
        <p><strong>Test Duration:</strong> ${Math.round(data.state.testRunDurationMs / 1000)}s</p>
        <p><strong>Virtual Users:</strong> ${data.metrics.vus_max.values.max}</p>
    </div>
    
    <div class="metrics">
        <div class="metric">
            <h3>Requests</h3>
            <p>Total: ${data.metrics.http_reqs.values.count}</p>
            <p>Rate: ${Math.round(data.metrics.http_reqs.values.rate * 100) / 100}/s</p>
        </div>
        
        <div class="metric">
            <h3>Response Time</h3>
            <p>Average: ${Math.round(data.metrics.http_req_duration.values.avg)}ms</p>
            <p>95th percentile: ${Math.round(data.metrics.http_req_duration.values['p(95)'])}ms</p>
        </div>
        
        <div class="metric">
            <h3>Error Rate</h3>
            <p class="${data.metrics.http_req_failed.values.rate > 0.1 ? 'fail' : 'pass'}">
                ${Math.round(data.metrics.http_req_failed.values.rate * 10000) / 100}%
            </p>
        </div>
    </div>
    
    <h2>Threshold Results</h2>
    <table>
        <tr><th>Metric</th><th>Threshold</th><th>Result</th></tr>
        ${Object.entries(data.metrics).map(([name, metric]) => {
          if (metric.thresholds) {
            return Object.entries(metric.thresholds).map(([threshold, result]) => 
              `<tr>
                <td>${name}</td>
                <td>${threshold}</td>
                <td class="${result.ok ? 'pass' : 'fail'}">${result.ok ? 'PASS' : 'FAIL'}</td>
              </tr>`
            ).join('');
          }
          return '';
        }).join('')}
    </table>
</body>
</html>`;
}