async def crunch(items):
    total = 0
    for x in items:
        total += x * x
    return total
