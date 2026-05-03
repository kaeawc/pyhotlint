def collect(tensors):
    out = []
    for t in tensors:
        out.append(t.cpu())
    return out
