from prometheus_client import Counter as Ctr

# Aliased import: the binding is `Ctr`, not `Counter`. Must still fire
# because `Ctr` IS the prometheus_client Counter.
SOMETHING = Ctr("hits", "Hit count", labelnames=("region",))
