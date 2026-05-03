import torch


@torch.no_grad()
def predict(model, x):
    model.eval()
    return model(x)


@torch.inference_mode()
def predict_strict(model, x):
    model.eval()
    return model(x)
