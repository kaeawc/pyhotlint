def crunch(items):
    # Plain sync function — no event loop to block.
    total = 0
    for x in items:
        total += x * x
    return total
