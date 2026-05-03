def add_features(x):
    a = x.cpu()
    b = x.cuda()
    return a + b
