def collect(tensors, device):
    tensors = [t.to(device) for t in tensors]  # ok: list-comp is not a `for_statement`
    big = tensors[0].cpu()
    for t in tensors:
        # No host/device copy inside the loop.
        big = big + t
    return big
