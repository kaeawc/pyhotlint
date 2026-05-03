def run(engine, expr):
    # `engine.eval(expr)` is a different API — non-zero arity, not a model.
    return engine.eval(expr)


def configure(parser, source):
    return parser.eval(source, mode="strict")
