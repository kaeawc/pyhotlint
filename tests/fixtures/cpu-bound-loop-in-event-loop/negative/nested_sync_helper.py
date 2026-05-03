async def handler(items):
    def crunch():
        # Sync helper — its loop is its own concern.
        total = 0
        for x in items:
            total += x
        return total

    return crunch()
