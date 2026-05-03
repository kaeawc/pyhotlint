import threading

lock = threading.Lock()


async def handler():
    with lock:
        # No await inside; lock acquire/release is fine.
        x = compute()
    return x


def compute():
    return 1
