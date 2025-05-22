import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.1.0/index.js';

const voteRate = new Rate('vote_rate');
const skipRate = new Rate('skip_rate');
const feedLatency = new Trend('feed_latency');
const voteLatency = new Trend('vote_latency');
const skipLatency = new Trend('skip_latency');

export const options = {
  stages: [
    { duration: '30s', target: 5 },
    { duration: '1m', target: 5 },
    { duration: '30s', target: 10 },
    { duration: '1m', target: 10 },
    { duration: '30s', target: 20 },
    { duration: '1m', target: 20 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'],
    'vote_rate': ['rate>0.95'],
    'skip_rate': ['rate>0.95'],
    'feed_latency': ['p(95)<1000'],
    'vote_latency': ['p(95)<200'],
    'skip_latency': ['p(95)<200'],
  },
};

const BASE_URL = 'http://localhost:8080/api';
const TAGS = ['sports', 'news', 'entertainment', 'technology', 'politics'];

// Test user credentials
const TEST_USER = {
  email: 'test@example.com',
  password: 'password123'
};

// Function to get JWT token
function getAuthToken() {
  const loginPayload = JSON.stringify({
    email: TEST_USER.email,
    password: TEST_USER.password
  });

  const loginResponse = http.post(`${BASE_URL}/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' }
  });

  const loginCheck = check(loginResponse, {
    'login successful': (r) => r.status === 200,
  });

  if (!loginCheck) {
    console.error('Login failed:', loginResponse.status, loginResponse.body);
    return null;
  }

  try {
    const token = loginResponse.json('token');
    if (!token) {
      console.error('No token in response:', loginResponse.body);
      return null;
    }
    return token;
  } catch (e) {
    console.error('Failed to parse login response:', e);
    return null;
  }
}

function getRandomTag() {
  return TAGS[Math.floor(Math.random() * TAGS.length)];
}

function createPoll(token) {
  const payload = JSON.stringify({
    title: `Test Poll ${randomString(10)}`,
    options: ['Option 1', 'Option 2', 'Option 3'],
    tags: [getRandomTag()],
  });

  const response = http.post(`${BASE_URL}/polls`, payload, {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
  });

  const createCheck = check(response, {
    'create poll status is 201': (r) => r.status === 201,
  });

  if (!createCheck) {
    console.error('Create poll failed:', response.status, response.body);
    return null;
  }

  try {
    const pollId = response.json('poll_id');
    if (!pollId) {
      console.error('No poll_id in response:', response.body);
      return null;
    }
    return pollId;
  } catch (e) {
    console.error('Failed to parse create poll response:', e);
    return null;
  }
}

function getFeed(token) {
  const startTime = new Date();
  const response = http.get(`${BASE_URL}/polls?tag=${getRandomTag()}&page=1&limit=10`, {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  const endTime = new Date();
  feedLatency.add(endTime - startTime);

  const feedCheck = check(response, {
    'get feed status is 200': (r) => r.status === 200,
  });

  if (!feedCheck) {
    console.error('Get feed failed:', response.status, response.body);
    return null;
  }

  try {
    const data = response.json('data');
    if (!data || !data.polls) {
      console.error('Invalid feed response structure:', response.body);
      return null;
    }
    return data.polls;
  } catch (e) {
    console.error('Failed to parse feed response:', e);
    return null;
  }
}

function voteOnPoll(pollId, token) {
  if (!pollId) return false;

  const startTime = new Date();
  const payload = JSON.stringify({
    optionIndex: Math.floor(Math.random() * 3),
  });

  const response = http.post(`${BASE_URL}/polls/${pollId}/vote`, payload, {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
  });
  const endTime = new Date();
  voteLatency.add(endTime - startTime);

  const success = check(response, {
    'vote status is 200': (r) => r.status === 200,
  });
  voteRate.add(success);

  if (!success) {
    console.error('Vote failed:', response.status, response.body);
  }

  return success;
}

function skipPoll(pollId, token) {
  if (!pollId) return false;

  const startTime = new Date();
  const response = http.post(`${BASE_URL}/polls/${pollId}/skip`, null, {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
  });
  const endTime = new Date();
  skipLatency.add(endTime - startTime);

  const success = check(response, {
    'skip status is 200': (r) => r.status === 200,
  });
  skipRate.add(success);

  if (!success) {
    console.error('Skip failed:', response.status, response.body);
  }

  return success;
}

export default function () {
  // Get authentication token
  const token = getAuthToken();
  if (!token) {
    console.error('Failed to get authentication token');
    return;
  }

  // Create a new poll
  const pollId = createPoll(token);
  if (!pollId) {
    console.error('Failed to create poll, skipping iteration');
    return;
  }
  sleep(1);

  // Get feed and interact with polls
  const polls = getFeed(token);
  if (!polls || polls.length === 0) {
    console.error('No polls in feed, skipping interaction');
    return;
  }

  // Interact with a random poll from the feed
  const randomPoll = polls[Math.floor(Math.random() * polls.length)];
  if (Math.random() < 0.7) {
    voteOnPoll(randomPoll.id, token);
  } else {
    skipPoll(randomPoll.id, token);
  }

  sleep(1);
} 