import torch


def predict(model, x):
    with torch.inference_mode():
        model.eval()
        return model(x)
