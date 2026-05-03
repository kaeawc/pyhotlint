def add_features(x, b):
    a = x.cpu()
    # b's device is unknown; rule must not fire even though a is known cpu.
    return a + b
