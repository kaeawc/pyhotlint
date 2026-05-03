import asyncio

lock = asyncio.Lock()


async def handler():
    with lock:
        result = await fetch()
    return result


async def fetch():
    return 1
