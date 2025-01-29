import os


def bar():
    x = 1
    x += 1
    return x


def foo():
    y = 1
    while True:
        y += bar()


def simple():
    foo()


def main():
    print(f"Current process PID: {os.getpid()}")
    simple()
