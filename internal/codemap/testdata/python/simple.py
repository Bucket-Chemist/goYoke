import os
from collections import OrderedDict


def hello(name: str) -> str:
    return f"Hello, {name}!"


class MyClass:
    def __init__(self, value: int) -> None:
        self._value = value

    @property
    def value(self) -> int:
        return self._value

    def set_value(self, v: int) -> None:
        self._value = v


@staticmethod
def standalone_static() -> str:
    return "static"


def _private_func() -> None:
    pass


MY_CONST = 42
