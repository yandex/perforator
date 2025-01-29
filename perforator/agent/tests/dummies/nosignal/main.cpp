#include <csignal>

void ignore(int) {
    // pass
}

int main() {
    int signals[]{SIGTERM, SIGINT, SIGABRT, SIGSEGV};
    for (int signo : signals) {
        signal(signo, ignore);
    }

    volatile bool always = true;
    while (always) {
        // pass
    }
}
