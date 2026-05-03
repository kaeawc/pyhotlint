import time


async def handler():
    time.sleep(0.5)  # blocks the event loop
    return "ok"
