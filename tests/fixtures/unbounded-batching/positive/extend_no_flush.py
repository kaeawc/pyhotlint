class Aggregator:
    def __init__(self):
        self.events = []

    async def ingest(self, batch):
        self.events.extend(batch)
        return None
