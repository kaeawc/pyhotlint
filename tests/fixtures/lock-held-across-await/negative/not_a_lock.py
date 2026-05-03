async def handler(path):
    # `open` returns a file handle, not a lock; with-statement is fine.
    with open(path, "rb") as f:
        data = await read_async(f)
    return data


async def read_async(f):
    return f.read()
