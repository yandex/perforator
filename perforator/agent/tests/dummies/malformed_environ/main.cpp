#include <cstring>

extern char** environ;

void clobber_environ() {
    char** envp = environ;
    while (char* env = *envp++) {
        size_t len = strlen(env);
        memset(env, 0xff, len);
        env[len] = 0xfe;
    }
}

void loop() {
    volatile bool running = true;
    while (running) {}
}

int main() {
    clobber_environ();
    loop();
}
