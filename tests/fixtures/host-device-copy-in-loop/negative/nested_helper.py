def collect(tensors):
    def to_host(t):
        # Sync helper — its .cpu() is its own concern, not the outer loop's.
        return t.cpu()

    out = []
    for t in tensors:
        out.append(to_host(t))
    return out
