package tech.perforator.unwind;

import java.util.concurrent.Executor;
import java.util.concurrent.Executors;

class Main {
    public static void main(String[] args) {
        for (int i = 0; i < 1; i++) {
            foo3();
        }
        var ex = Executors.newVirtualThreadPerTaskExecutor();
        vthreadNoSuspend(ex);
        vthreadSuspend(ex);
        ex.close();
    }

    static void foo3() {
        foo2();
    }

    static void foo2() {
        foo1();
    }

    static void foo1() {
        foo0();
    }

    static void foo0() {
        for (int i = 0; i < 100000; i++) {
            new Main().bar(100000 - i - 1);
        }
    }

    void bar(int x) {
        Native.unwindIfZero0(x);
    }


    static void vthreadNoSuspend(Executor ex) {
        ex.execute(() -> {
            foo3();
        });
    }

    static void vthreadSuspend(Executor ex) {
        ex.execute(() -> {
            Thread.yield();
            foo3();
        });
    }
}
