def stream(batches, device):
    while batches:
        batch = batches.pop()
        yield batch.to(device)
