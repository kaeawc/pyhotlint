def to_dicts(records):
    out = []
    for r in records:
        # Zero-arg .to() is not a device copy (likely a different API).
        out.append(r.to())
    return out
