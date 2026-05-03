import torch


def predict(model, x):
    with torch.no_grad():
        model.eval()
        return model(x)
