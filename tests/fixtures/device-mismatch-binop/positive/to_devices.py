def matmul(x, y):
    a = x.to("cpu")
    b = y.to("cuda:0")
    return a * b
