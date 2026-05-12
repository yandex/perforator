void call(int n, void (*f)(void)) {
    for (int i = 0; i < n; i++) {
        f();
    }
}

void loop(int n) {
    for (int i = 0; i < n; i++) {
        /* prevent compiler optimizations from skipping loop entirely */
        __asm__("");
    }
}

void func(void) {
}
