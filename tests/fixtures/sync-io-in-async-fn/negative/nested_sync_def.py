import time


async def handler():
    def helper():
        # Nested sync function — sync I/O inside is fine, the helper
        # is the unit that may be scheduled separately.
        time.sleep(0.1)

    return helper
