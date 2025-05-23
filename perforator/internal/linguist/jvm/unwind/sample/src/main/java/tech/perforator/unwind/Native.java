package tech.perforator.unwind;

class Native {
    static {
        System.loadLibrary("unwind-jni");
    }

    private Native() {
    }

    static native void unwind0();
    static native void unwindIfZero0(int x);
}
