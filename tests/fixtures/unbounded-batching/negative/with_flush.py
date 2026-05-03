class Server:
    def __init__(self):
        self.batch = []

    async def handle(self, item):
        self.batch.append(item)
        if len(self.batch) >= 32:
            await self.flush_batch()

    async def flush_batch(self):
        self.batch.clear()
