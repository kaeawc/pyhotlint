def predict(model, x):
    model.eval()
    return model(x)
