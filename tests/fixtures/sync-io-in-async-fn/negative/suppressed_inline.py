import time


async def handler():
    time.sleep(1)  # pyhotlint: ignore[sync-io-in-async-fn]
    return "ok"
