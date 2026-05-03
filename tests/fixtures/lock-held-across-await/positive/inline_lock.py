import threading


async def handler():
    with threading.Lock():
        result = await fetch()
    return result


async def fetch():
    return 1
