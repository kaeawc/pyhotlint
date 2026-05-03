async def matmul(a, b):
    out = []
    for row in a:
        for val in row:
            out.append(val * 2)
    return out
