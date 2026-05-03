async def handler(items):
    # Local accumulator is scope-bounded; not the unbounded-batching shape.
    out = []
    for item in items:
        out.append(item * 2)
    return out
