from transformers import AutoTokenizer, AutoModel


def load_tokenizer():
    return AutoTokenizer.from_pretrained("bert-base-uncased")


def load_model():
    # Different function — pyhotlint cannot prove these are paired,
    # so does not flag. (Cross-scope tracking is a future capability.)
    return AutoModel.from_pretrained("roberta-base")
