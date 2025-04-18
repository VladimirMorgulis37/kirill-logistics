// src/config.js
const API_URLS = {
    auth: process.env.REACT_APP_AUTH_URL || "http://localhost:8081",
    orders: process.env.REACT_APP_ORDERS_URL || "http://localhost:8082",
    tracking: process.env.REACT_APP_TRACKING_URL || "http://localhost:8083",
    analytics: process.env.REACT_APP_ANALYTICS_URL || "http://localhost:8084",
  };

  export default API_URLS;
