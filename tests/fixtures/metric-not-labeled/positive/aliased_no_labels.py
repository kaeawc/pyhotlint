from prometheus_client import Counter as Ctr

# `Ctr` resolves to prometheus_client.Counter; missing labelnames must
# still be flagged.
HITS = Ctr("hits_total", "Total hits")
