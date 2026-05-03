from prometheus_client import Counter, Gauge

REQUESTS = Counter("requests_total", "Total requests", labelnames=("service", "route"))
INFLIGHT = Gauge("inflight", "Inflight requests", labelnames=["service"])
