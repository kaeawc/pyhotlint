import time


def handler():
    # Plain sync function — sync I/O is fine here.
    time.sleep(0.5)
    return "ok"
