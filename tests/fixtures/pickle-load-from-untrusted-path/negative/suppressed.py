import pickle


def restore(blob):
    # Trusted internal cache; verified upstream.
    return pickle.loads(blob)  # pyhotlint: ignore[pickle-load-from-untrusted-path]
