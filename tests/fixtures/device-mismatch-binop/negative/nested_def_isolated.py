def outer(x):
    a = x.cpu()

    def inner():
        # Inner has its own scope; outer's `a = cpu` does not leak.
        # The binop here uses inner's untracked locals only.
        c = x
        d = x
        return c + d

    return a, inner
