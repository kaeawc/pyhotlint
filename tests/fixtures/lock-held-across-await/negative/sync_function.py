import threading

lock = threading.Lock()


def handler():
    # Sync function — there is no await, so the rule does not apply.
    with lock:
        return 42
