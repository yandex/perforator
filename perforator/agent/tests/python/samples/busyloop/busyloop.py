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
    print("Current process PID: {0}".format(os.getpid()))
    simple()


if __name__ == "__main__":
    main()
