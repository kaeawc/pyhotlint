import pickle


def load_weights(path):
    with open(path, "rb") as f:
        return pickle.load(f)
