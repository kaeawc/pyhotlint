def fuse(x, y):
    return x.cpu() + y.cuda()
