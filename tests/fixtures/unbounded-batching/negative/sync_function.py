class Server:
    def __init__(self):
        self.batch = []

    def handle(self, item):
        # Sync function — out of scope for the rule.
        self.batch.append(item)
