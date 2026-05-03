import prometheus_client as pc

LATENCY = pc.Histogram("latency_ms", "Request latency in ms")
