#include <iostream>
#include <thread>
#include <chrono>
#include <vector>
#include <functional>
#include <unistd.h>
#include <dlfcn.h>
#include <mutex>
#include <sys/syscall.h>

typedef int (*TargetFuncLevel2Type)(int);
typedef int (*RecursiveFuncType)(int);
typedef int (*LambdaCallerType)(int);

// Define gettid() using syscall
pid_t gettid() {
    return syscall(SYS_gettid);
}

__attribute__((noinline)) int target_function_level1(int value, TargetFuncLevel2Type target_function_level2) {
    return target_function_level2(value) + 100;
}

int main(int, char** argv) {
    std::cout << "Uprobe test program started" << std::endl;
    std::cout << "Process ID: " << getpid() << std::endl;

    void* lib_handle = dlopen(argv[1], RTLD_LAZY);
    if (!lib_handle) {
        std::cerr << "Error loading library: " << dlerror() << std::endl;
        return 1;
    }

    auto target_function_level2 = (TargetFuncLevel2Type)dlsym(lib_handle, "target_function_level2");
    auto recursive_function = (RecursiveFuncType)dlsym(lib_handle, "recursive_function");
    auto lambda_caller = (LambdaCallerType)dlsym(lib_handle, "lambda_caller");

    const char* dlsym_error = dlerror();
    if (dlsym_error) {
        std::cerr << "Error resolving symbols: " << dlsym_error << std::endl;
        dlclose(lib_handle);
        return 1;
    }

    std::mutex print_mutex;
    auto thread_func = [&](int thread_num) {
        {
            std::lock_guard<std::mutex> lock(print_mutex);
            std::cout << "Thread " << thread_num << " started. "
                     << "PID: " << getpid()
                     << " TID: " << gettid() << std::endl;
        }

        int iteration = 0;
        while (true) {
            iteration++;

            int value = iteration % 5 + 1;
            target_function_level1(value, target_function_level2);
            recursive_function(value);
            lambda_caller(value);

            std::this_thread::sleep_for(std::chrono::seconds(1));
        }
    };

    std::vector<std::thread> threads;
    for (int i = 0; i < 3; i++) {
        threads.emplace_back(thread_func, i);
    }

    for (auto& t : threads) {
        t.join();
    }

    dlclose(lib_handle);
    return 0;
}
