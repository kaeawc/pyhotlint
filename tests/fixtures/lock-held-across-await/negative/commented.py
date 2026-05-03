import threading

lock = threading.Lock()


async def handler():
    with lock:  # intentional: cooperative yield to background flush
        result = await fetch()
    return result


async def fetch():
    return 1
