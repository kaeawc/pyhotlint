class Counter:
    def __init__(self, name, doc):
        self.name = name
        self.doc = doc


# Not a prometheus_client Counter — must not fire.
hits = Counter("hits", "Hits handled")
