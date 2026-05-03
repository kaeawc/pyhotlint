from prometheus_client import Counter

REQUESTS = Counter("requests_total", "Total requests handled")
