class Server:
    def __init__(self):
        self.batch = []

    async def handle(self, item):
        self.batch.append(item)
        return "queued"
