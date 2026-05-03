import threading

lock = threading.Lock()


async def handler():
    with lock:
        result = await fetch_remote()
    return result


async def fetch_remote():
    return 42
